package registry

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"railyard/internal/config"
	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestFilterSemverVersions(t *testing.T) {
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	filtered := reg.filterSemverVersions([]types.VersionInfo{
		{Version: "1.2.3"},
		{Version: "v2.3.4"},
		{Version: "1.2"},
		{Version: "1.2.3-beta.1"},
		{Version: "1.2.3+build.1"},
		{Version: "not-semver"},
		{Version: ""},
	}, "test")

	require.Len(t, filtered, 2)
	require.Equal(t, "1.2.3", filtered[0].Version)
	require.Equal(t, "v2.3.4", filtered[1].Version)
}

func TestGetGitHubVersionsAuthFallbackAndCache(t *testing.T) {
	cfg := config.NewConfig(testutil.TestLogSink{})
	updated := cfg.UpdateGithubToken("github_pat_test_token")
	require.Equal(t, types.ResponseSuccess, updated.Status)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	originalBaseURL := registryGitHubAPIBaseURL
	t.Cleanup(func() {
		registryGitHubAPIBaseURL = originalBaseURL
	})

	var requestCount int32
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&requestCount, 1)
		require.Equal(t, "/repos/owner/repo/releases", r.URL.Path)

		// First authenticated request fails with 401 to trigger fallback.
		if current == 1 {
			require.Equal(t, "Bearer github_pat_test_token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		require.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"tag_name":"v1.2.3","name":"v1.2.3","body":"notes","prerelease":false,"published_at":"2026-01-01T00:00:00Z","assets":[]}]`)
	}))
	defer server.Close()

	registryGitHubAPIBaseURL = server.URL
	versions, err := reg.GetVersions("github", "owner/repo")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, "v1.2.3", versions[0].Version)
	require.EqualValues(t, 2, atomic.LoadInt32(&requestCount))

	// Second call should be served from in-memory cache.
	versions, err = reg.GetVersions("github", "owner/repo")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.EqualValues(t, 2, atomic.LoadInt32(&requestCount))
}

func TestClearVersionsCache(t *testing.T) {
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	reg.setCachedVersions("github|owner/repo", []types.VersionInfo{{Version: "v1.0.0"}})

	_, ok := reg.getCachedVersions("github|owner/repo")
	require.True(t, ok)

	reg.clearVersionsCache()

	_, ok = reg.getCachedVersions("github|owner/repo")
	require.False(t, ok)
}

func TestGetVersionsResponseSetsGitHubAuthErrorType(t *testing.T) {
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	originalBaseURL := registryGitHubAPIBaseURL
	t.Cleanup(func() {
		registryGitHubAPIBaseURL = originalBaseURL
	})

	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/releases", r.URL.Path)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	registryGitHubAPIBaseURL = server.URL
	response := reg.GetVersionsResponse("github", "owner/repo")

	require.Equal(t, types.ResponseError, response.Status)
	require.Equal(t, types.VersionsErrorGitHubAuth, response.ErrorType)
	require.Contains(t, response.Message, types.GitHubTokenDocsURL)
}

func TestGetVersionsResponseSetsGitHubFetchErrorType(t *testing.T) {
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	originalBaseURL := registryGitHubAPIBaseURL
	t.Cleanup(func() {
		registryGitHubAPIBaseURL = originalBaseURL
	})

	// Port 1 on localhost should be closed; this forces a fetch/connect error.
	registryGitHubAPIBaseURL = "http://127.0.0.1:1"
	response := reg.GetVersionsResponse("github", "owner/repo")

	require.Equal(t, types.ResponseError, response.Status)
	require.Equal(t, types.VersionsErrorGitHubFetch, response.ErrorType)
	require.Contains(t, response.Message, "failed to fetch")
}
