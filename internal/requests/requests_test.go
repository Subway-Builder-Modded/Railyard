package requests

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestIsGitHubHost(t *testing.T) {
	require.True(t, IsGitHubHost("github.com"))
	require.True(t, IsGitHubHost("api.github.com"))
	require.True(t, IsGitHubHost("raw.githubusercontent.com"))
	require.False(t, IsGitHubHost("example.com"))
}

func TestGetWithGithubTokenAppliesHeadersAndToken(t *testing.T) {
	var seenAuth string
	var seenUA string
	var seenCustom string

	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenUA = r.Header.Get("User-Agent")
		seenCustom = r.Header.Get("X-Test")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := GetWithGithubToken(NewAPIClient(), GithubTokenRequestArgs{
		URL:              server.URL,
		GitHubToken:      "token-abc",
		Context:          context.Background(),
		Headers:          map[string]string{"X-Test": "1"},
		ForceAuthByToken: true,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "Bearer token-abc", seenAuth)
	require.Equal(t, "Railyard-Desktop-App", seenUA)
	require.Equal(t, "1", seenCustom)
}

func TestGetWithGithubTokenFallsBackToUnauthenticatedOn401(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0
	authHeaders := make([]string, 0, 2)
	callbackCodes := make([]int, 0, 1)

	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		index := requestCount
		mu.Unlock()

		if index == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := GetWithGithubToken(NewAPIClient(), GithubTokenRequestArgs{
		URL:              server.URL,
		GitHubToken:      "token-abc",
		ForceAuthByToken: true,
		OnTokenRejected: func(statusCode int) {
			callbackCodes = append(callbackCodes, statusCode)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []int{http.StatusUnauthorized}, callbackCodes)
	require.Len(t, authHeaders, 2)
	require.Equal(t, "Bearer token-abc", authHeaders[0])
	require.Empty(t, authHeaders[1])
}

func TestGetWithGithubTokenSkipsAuthForNonGitHubHostWhenNotForced(t *testing.T) {
	seenAuth := ""
	server := testutil.NewLocalhostServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := GetWithGithubToken(NewAPIClient(), GithubTokenRequestArgs{
		URL:         server.URL,
		GitHubToken: "token-abc",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, seenAuth)
}

func TestAPIStatusErrorIncludesDocsForGitHubForbidden(t *testing.T) {
	err := NewAPIStatusError(APISourceGitHub, http.StatusForbidden, "owner/repo")
	require.Error(t, err)
	require.Contains(t, err.Error(), types.GitHubTokenDocsURL)
}
