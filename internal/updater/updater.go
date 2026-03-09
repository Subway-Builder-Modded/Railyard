package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/types"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// launchElevated starts exePath with UAC elevation prompt.
func launchElevated(exePath string, args string, workingDir string) error {
	verb := utf16Ptr("runas") // request elevation
	file := utf16Ptr(exePath)

	var params *uint16
	if args != "" {
		params = utf16Ptr(args)
	}

	var dir *uint16
	if workingDir != "" {
		dir = utf16Ptr(workingDir)
	}

	// HINSTANCE > 32 means success. <= 32 is an error code.
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(dir)),
		1, // SW_SHOWNORMAL
	)

	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW failed with code %d", ret)
	}
	return nil
}

func deleteOldTempInstallers() error {
	files, err := os.ReadDir(os.TempDir())
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "railyard-installer-") {
			err := os.Remove(path.Join(os.TempDir(), file.Name()))
			if err != nil {
				fmt.Printf("Failed to delete old installer %s: %v\n", file.Name(), err)
			}
		}
	}
	return nil
}

func CheckForUpdates(ctx context.Context, progressFunc types.ProgressFunc, log logger.Logger) error {
	err := deleteOldTempInstallers()
	if err != nil {
		fmt.Printf("Error cleaning up old installers: %v\n", err)
	}
	versions, err := pullReleases(log)
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return err
	}

	for _, v := range versions {
		if VersionIsNewerThanInstalled(v.Version) {
			fmt.Printf("New version available: %s\n", v.Version)
			result, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type:    wailsruntime.QuestionDialog,
				Title:   "Update Available",
				Message: fmt.Sprintf("Version %s of Railyard is available. Would you like to download and install it?", v.Version),
				Buttons: []string{"Yes", "No"},
			})
			if err != nil {
				fmt.Printf("Error showing update dialog: %v\n", err)
				return err
			}
			if result == "Yes" {
				var downloadURL string
				switch runtime.GOOS {
				case "windows":
					if runtime.GOARCH == "amd64" && v.WindowsX64DownloadURL != "" {
						downloadURL = v.WindowsX64DownloadURL
					}
					if runtime.GOARCH == "arm64" && v.WindowsARMDownloadURL != "" {
						downloadURL = v.WindowsARMDownloadURL
					}
				case "darwin":
					downloadURL = v.MacOSDownloadURL
				case "linux":
					downloadURL = v.LinuxDownloadURL
				}
				if downloadURL == "" {
					fmt.Printf("No suitable installer found for this platform in version %s\n", v.Version)
					return fmt.Errorf("no suitable installer found for this platform in version %s", v.Version)
				}
				err = downloadAndRunInstaller(downloadURL, ctx, progressFunc)
				if err != nil {
					fmt.Printf("Error downloading or running installer: %v\n", err)
					return err
				}
			}
			break
		}
	}
	return nil
}

func downloadAndRunInstaller(downloadURL string, ctx context.Context, downloadProgress types.ProgressFunc) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download installer from %q: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download installer from %q: status code %d", downloadURL, resp.StatusCode)
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "railyard-installer-*"+path.Ext(downloadURL))
	if err != nil {
		return fmt.Errorf("failed to create temp file for installer: %w", err)
	}

	// Wrap the response body in a progress reader to report download progress
	progressReader := &types.ProgressReader{
		Reader:     resp.Body,
		Total:      resp.ContentLength,
		OnProgress: downloadProgress,
		ItemId:     "installer",
	}

	_, err = io.Copy(tempFile, progressReader)
	if err != nil {
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to save installer to temp file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if runtime.GOOS == "linux" {
		err = os.Chmod(tempFile.Name(), 0755)
		if err != nil {
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to make installer executable: %w", err)
		}
		proc, err := os.StartProcess(tempFile.Name(), []string{tempFile.Name()}, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{},
		})
		if err != nil {
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to start installer: %w", err)
		}
		proc.Release()

	}
	if runtime.GOOS == "windows" {
		err = launchElevated(tempFile.Name(), "", "")
		if err != nil {
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to launch installer with elevation: %w", err)
		}
	}
	if runtime.GOOS == "darwin" {
		// For DMG files, we can use the "open" command to launch it, which will handle mounting and running the installer inside.
		proc, err := os.StartProcess("/usr/bin/open", []string{"/usr/bin/open", tempFile.Name()}, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		})
		if err != nil {
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to start installer: %w", err)
		}
		proc.Release()
	}
	wailsruntime.Quit(ctx)
	return nil
}

func VersionIsNewerThanInstalled(version string) bool {
	installed := constants.RAILYARD_VERSION
	// Strip leading "v" if present for comparison
	installed = strings.TrimPrefix(installed, "v")
	version = strings.TrimPrefix(version, "v")

	newVersionIsRC := strings.Contains(version, "rc")
	installedVersionIsRC := strings.Contains(installed, "rc")
	if newVersionIsRC && !installedVersionIsRC {
		return false
	}

	// Trim pattern +rc.x if present for comparison
	if idx := strings.Index(version, "+rc."); idx >= 0 {
		version = version[:idx]
	}
	if idx := strings.Index(installed, "+rc."); idx >= 0 {
		installed = installed[:idx]
	}

	installedParts := strings.Split(installed, ".")
	versionParts := strings.Split(version, ".")

	for i := 0; i < len(installedParts) && i < len(versionParts); i++ {
		if versionParts[i] > installedParts[i] {
			return true
		}
	}
	return false
}

func pullReleases(log logger.Logger) ([]types.RailyardVersionInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", constants.RAILYARD_REPO)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Railyard-Desktop-App")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub releases for %q: %w", constants.RAILYARD_REPO, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for %q", resp.StatusCode, constants.RAILYARD_REPO)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	releases, err := files.ParseJSON[[]types.GithubRelease](body, "releases")
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub releases JSON: %w", err)
	}

	versions := make([]types.RailyardVersionInfo, 0, len(releases))
	for _, rel := range releases {
		v := types.RailyardVersionInfo{
			Version:    rel.TagName,
			Name:       rel.Name,
			Changelog:  rel.Body,
			Date:       rel.PublishedAt,
			Prerelease: rel.Prerelease,
		}
		for _, asset := range rel.Assets {
			if strings.Contains(asset.Name, "amd64.AppImage") {
				v.LinuxDownloadURL = asset.BrowserDownloadURL
			}
			if strings.Contains(asset.Name, "macos-universal.dmg") {
				v.MacOSDownloadURL = asset.BrowserDownloadURL
			}
			if strings.Contains(asset.Name, "amd64-installer.exe") {
				v.WindowsX64DownloadURL = asset.BrowserDownloadURL
			}
			if strings.Contains(asset.Name, "arm64-installer.exe") {
				v.WindowsARMDownloadURL = asset.BrowserDownloadURL
			}
		}
		if len(returnMissingDownloads(v)) != 0 {
			log.Warn("Release %s is missing downloads for: %s", v.Version, strings.Join(returnMissingDownloads(v), ", "))
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func returnMissingDownloads(version types.RailyardVersionInfo) []string {
	missing := []string{}
	if version.LinuxDownloadURL == "" {
		missing = append(missing, "linux")
	}

	if version.MacOSDownloadURL == "" {
		missing = append(missing, "macos")
	}

	if version.WindowsX64DownloadURL == "" {
		missing = append(missing, "windows-x64")
	}

	if version.WindowsARMDownloadURL == "" {
		missing = append(missing, "windows-arm")
	}
	return missing
}
