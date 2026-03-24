package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/requests"
	"railyard/internal/types"
	"runtime"
	"strconv"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var updaterGitHubAPIBaseURL = types.GitHubAPIBaseURL
var updaterHTTPClient = requests.NewAPIClient()
var updaterDownloadHTTPClient = requests.NewDownloadClient()

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

func CheckForUpdates(ctx context.Context, progressFunc types.ProgressFunc, log logger.Logger, githubToken string) error {
	log.Info("Checking for updates")
	err := deleteOldTempInstallers()
	if err != nil {
		log.Error("Failed to delete old temp installers", err)
	}
	versions, err := pullReleases(log, githubToken)
	if err != nil {
		log.Error("Error checking for updates", err)
		return err
	}

	for _, v := range versions {
		if VersionIsNewerThanInstalled(v.Version) {
			log.Info("New version available", "version", v.Version)
			result, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type:    wailsruntime.QuestionDialog,
				Title:   "Update Available",
				Message: fmt.Sprintf("Version %s of Railyard is available. Would you like to download and install it?", v.Version),
				Buttons: []string{"Yes", "No"},
			})
			if err != nil {
				log.Error("Error showing update dialog", err)
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
					downloadURL = v.LinuxCurrentDownloadURL
				}
				if downloadURL == "" {
					log.Error("No suitable installer found for this platform", err, "version", v.Version)
					return fmt.Errorf("no suitable installer found for this platform in version %s", v.Version)
				}
				err = downloadAndRunInstaller(downloadURL, ctx, progressFunc, log)
				if err != nil {
					log.Error("Error downloading or running installer", err)
					return err
				}
			}
			break
		}
	}
	return nil
}

func downloadAndRunInstaller(downloadURL string, ctx context.Context, downloadProgress types.ProgressFunc, log logger.Logger) error {
	log.Info("Downloading installer", "url", downloadURL)
	resp, err := requests.GetWithGithubToken(updaterDownloadHTTPClient, requests.GithubTokenRequestArgs{
		URL: downloadURL,
	})
	if err != nil {
		log.Error("Failed to download installer", err)
		return fmt.Errorf("failed to download installer from %q: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("GitHub API returned non-200 status code when downloading installer", fmt.Errorf("status code %d", resp.StatusCode), "url", downloadURL)
		return fmt.Errorf("failed to download installer from %q: status code %d", downloadURL, resp.StatusCode)
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "railyard-installer-*"+path.Ext(downloadURL))
	if err != nil {
		log.Error("Failed to create temp file for installer", err)
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
		log.Error("Failed to save installer to temp file", err)
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to save installer to temp file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		os.Remove(tempFile.Name())
		log.Error("Failed to close temp file for installer", err)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	log.Info("Installer downloaded successfully, launching installer", "path", tempFile.Name())
	if runtime.GOOS == "linux" {
		proc := exec.Command("flatpak-spawn", "--host", "flatpak", "--user", "install", "--assumeyes", tempFile.Name())
		err := proc.Start()
		if err != nil {
			log.Error("Failed to start installer process", err)
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to start installer: %w", err)
		}
		proc.Process.Release()
	}
	if runtime.GOOS == "windows" {
		err = launchElevated(tempFile.Name(), "", "")
		if err != nil {
			log.Error("Failed to launch installer with elevation", err)
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
			log.Error("Failed to start installer process", err)
			os.Remove(tempFile.Name())
			return fmt.Errorf("failed to start installer: %w", err)
		}
		proc.Release()
	}
	log.Info("Installer launched successfully, exiting app")
	wailsruntime.Quit(ctx)
	return nil
}

func VersionIsNewerThanInstalled(version string) bool {
	installed := constants.RAILYARD_VERSION
	installed = strings.TrimPrefix(installed, "v")
	version = strings.TrimPrefix(version, "v")

	newVersionIsRC := strings.Contains(version, "rc")
	installedVersionIsRC := strings.Contains(installed, "rc")
	if newVersionIsRC && !installedVersionIsRC {
		return false
	}

	// Get rid of +rc but keep the RC number so we can check if one RC is newer than another
	version = strings.ReplaceAll(version, "+rc", "")
	installed = strings.ReplaceAll(installed, "+rc", "")

	installedParts := strings.Split(installed, ".")
	versionParts := strings.Split(version, ".")

	for i := 0; i < len(installedParts) && i < len(versionParts); i++ {
		versionPart, err1 := strconv.Atoi(versionParts[i])
		installedPart, err2 := strconv.Atoi(installedParts[i])
		if err1 != nil || err2 != nil {
			return false
		}
		if versionPart > installedPart {
			return true
		}
		if versionPart < installedPart {
			return false
		}
	}
	return false
}

func pullReleases(log logger.Logger, githubToken string) ([]types.RailyardVersionInfo, error) {
	log.Info("Fetching releases from GitHub")
	baseURL := strings.TrimRight(updaterGitHubAPIBaseURL, "/")
	apiURL := fmt.Sprintf("%s/repos/%s/releases", baseURL, constants.RAILYARD_REPO)
	resp, err := requests.GetWithGithubToken(updaterHTTPClient, requests.GithubTokenRequestArgs{
		URL:              apiURL,
		GitHubToken:      githubToken,
		ForceAuthByToken: true,
		Headers: map[string]string{
			"Accept": "application/vnd.github+json",
		},
		OnTokenRejected: func(statusCode int) {
			log.Warn("GitHub token rejected during updater check; retrying unauthenticated request", "status", statusCode)
		},
	})
	if err != nil {
		log.Error("Failed to fetch GitHub releases", err)
		return nil, fmt.Errorf("failed to fetch GitHub releases for %q: %w", constants.RAILYARD_REPO, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		log.Error("GitHub API returned non-200 status code when fetching releases", fmt.Errorf("status code %d", resp.StatusCode), "repo", constants.RAILYARD_REPO)
		return nil, fmt.Errorf("GitHub API returned status %d for %q", resp.StatusCode, constants.RAILYARD_REPO)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		log.Error("Failed to read GitHub API response", err)
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	releases, err := files.ParseJSON[[]types.GithubRelease](body, "releases")
	if err != nil {
		log.Error("Failed to parse GitHub releases JSON", err)
		return nil, fmt.Errorf("failed to parse GitHub releases JSON: %w", err)
	}

	log.Info("Parsing versions")
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
			if strings.Contains(asset.Name, "current-linux") {
				v.LinuxCurrentDownloadURL = asset.BrowserDownloadURL
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

	if version.LinuxCurrentDownloadURL == "" {
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
