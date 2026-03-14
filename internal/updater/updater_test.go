package updater

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"railyard/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestPullReleasesAuthFallback(t *testing.T) {
	originalBaseURL := updaterGitHubAPIBaseURL
	originalClient := updaterHTTPClient
	defer func() {
		updaterGitHubAPIBaseURL = originalBaseURL
		updaterHTTPClient = originalClient
	}()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&requestCount, 1)
		require.Equal(t, "/repos/Subway-Builder-Modded/Railyard/releases", r.URL.Path)

		if current == 1 {
			require.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		require.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"tag_name":"v1.2.3","name":"v1.2.3","body":"notes","prerelease":false,"published_at":"2026-01-01T00:00:00Z","assets":[]}]`)
	}))
	defer server.Close()

	updaterGitHubAPIBaseURL = server.URL
	updaterHTTPClient = server.Client()

	releases, err := pullReleases(testutil.TestLogSink{}, "ghp_test_token")
	require.NoError(t, err)
	require.Len(t, releases, 1)
	require.Equal(t, "v1.2.3", releases[0].Version)
	require.EqualValues(t, 2, atomic.LoadInt32(&requestCount))
}
