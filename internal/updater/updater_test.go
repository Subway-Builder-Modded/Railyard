package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/constants"
	"railyard/internal/logger"
	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestDeleteOldTempInstallers(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	t.Setenv("TMP", tempDir)
	t.Setenv("TEMP", tempDir)

	staleInstaller := filepath.Join(tempDir, "railyard-installer-old.exe")
	keepFile := filepath.Join(tempDir, "not-an-installer.txt")
	require.NoError(t, os.WriteFile(staleInstaller, []byte("installer"), 0o644))
	require.NoError(t, os.WriteFile(keepFile, []byte("keep"), 0o644))

	require.NoError(t, deleteOldTempInstallers())

	_, staleErr := os.Stat(staleInstaller)
	require.True(t, errors.Is(staleErr, fs.ErrNotExist))
	_, keepErr := os.Stat(keepFile)
	require.NoError(t, keepErr)
}

func TestVersionIsNewerThanInstalled(t *testing.T) {
	previousVersion := constants.RAILYARD_VERSION
	constants.RAILYARD_VERSION = "v1.0.0"
	t.Cleanup(func() {
		constants.RAILYARD_VERSION = previousVersion
	})

	require.False(t, VersionIsNewerThanInstalled(constants.RAILYARD_VERSION))
	require.True(t, VersionIsNewerThanInstalled("v9999.0.0"))
	require.False(t, VersionIsNewerThanInstalled("v1.0.1+rc1"))
}

func TestReturnMissingDownloads(t *testing.T) {
	empty := returnMissingDownloads(types.RailyardVersionInfo{})
	require.ElementsMatch(t, []string{"linux", "macos", "windows-x64", "windows-arm"}, empty)

	noneMissing := returnMissingDownloads(types.RailyardVersionInfo{
		LinuxCurrentDownloadURL: "linux",
		MacOSDownloadURL:        "mac",
		WindowsX64DownloadURL:   "win64",
		WindowsARMDownloadURL:   "winarm",
	})
	require.Empty(t, noneMissing)
}

func TestPullReleasesParsesResponse(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{
				"tag_name":"v1.2.3",
				"name":"Release 1.2.3",
				"body":"notes",
				"published_at":"2026-01-02T03:04:05Z",
				"prerelease":false,
				"assets":[
					{"name":"current-linux.tar.gz","browser_download_url":"https://example.com/linux"},
					{"name":"macos-universal.dmg","browser_download_url":"https://example.com/macos"},
					{"name":"amd64-installer.exe","browser_download_url":"https://example.com/win64"},
					{"name":"arm64-installer.exe","browser_download_url":"https://example.com/winarm"}
				]
			}
		]`)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_test.log"))
	versions, err := pullReleases(log, "")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, "v1.2.3", versions[0].Version)
	require.Equal(t, "https://example.com/linux", versions[0].LinuxCurrentDownloadURL)
	require.Equal(t, "https://example.com/macos", versions[0].MacOSDownloadURL)
	require.Equal(t, "https://example.com/win64", versions[0].WindowsX64DownloadURL)
	require.Equal(t, "https://example.com/winarm", versions[0].WindowsARMDownloadURL)
}

func TestPullReleasesRetriesWithoutTokenWhenRejected(t *testing.T) {
	var requests int
	var sawAuth bool

	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{
				"tag_name":"v2.0.0",
				"name":"Release 2.0.0",
				"body":"notes",
				"published_at":"2026-02-03T00:00:00Z",
				"prerelease":false,
				"assets":[]
			}
		]`)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_token_test.log"))
	versions, err := pullReleases(log, "bad-token")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.True(t, sawAuth)
	require.Equal(t, 2, requests, "expected token attempt and unauthenticated fallback")
}

func TestPullReleasesReturnsErrorOnBadStatus(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_bad_status.log"))
	_, err := pullReleases(log, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("status %d", http.StatusBadGateway))
}

func TestPullReleasesReturnsErrorOnInvalidJSON(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{ not-json`)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_bad_json.log"))
	_, err := pullReleases(log, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse GitHub releases JSON")
}

func TestDownloadAndRunInstallerReturnsErrorOnBadStatus(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_download_error.log"))
	err := downloadAndRunInstaller(server.URL+"/installer.exe", context.Background(), nil, log)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status code 503")
}

func TestCheckForUpdatesNoNewVersionReturnsNil(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{
				"tag_name":"v0.0.1",
				"name":"Release 0.0.1",
				"body":"notes",
				"published_at":"2026-01-02T03:04:05Z",
				"prerelease":false,
				"assets":[]
			}
		]`)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_check.log"))
	err := CheckForUpdates(context.Background(), nil, log, "")
	require.NoError(t, err)
}

func TestPullReleasesPropagatesFetchError(t *testing.T) {
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	previousBaseURL := updaterGitHubAPIBaseURL
	previousClient := updaterHTTPClient
	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()
	t.Cleanup(func() {
		updaterGitHubAPIBaseURL = previousBaseURL
		updaterHTTPClient = previousClient
	})

	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "updater_check_error.log"))
	_, err := pullReleases(log, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "GitHub API returned status")
}
