package requests

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"railyard/internal/types"
)

type GithubTokenRequestArgs struct {
	URL                    string
	GitHubToken            string
	Headers                map[string]string
	ForceAuthByToken         bool
	ShouldAuthenticateHost func(host string) bool
	OnTokenRejected        func(statusCode int)
}

// Generic client for API requests 
func NewAPIClient() *http.Client {
	return &http.Client{Timeout: types.RequestTimeout}
}

// Downloads can be larger and may require retries, so we use a custom client with more lenient timeouts and retry logic.
func NewDownloadClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   types.RequestTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   types.RequestTimeout,
			ResponseHeaderTimeout: types.RequestTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func IsGitHubHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(h, "github.com") || strings.Contains(h, "githubusercontent.com")
}

func GetWithGithubToken(client *http.Client, opts GithubTokenRequestArgs) (*http.Response, error) {
	shouldAuthenticate := opts.ShouldAuthenticateHost
	if shouldAuthenticate == nil {
		shouldAuthenticate = IsGitHubHost
	}

	buildRequest := func(withToken bool) (*http.Request, error) {
		req, err := http.NewRequest("GET", opts.URL, nil)
		if err != nil {
			return nil, err
		}

		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}
		req.Header.Set("User-Agent", types.RequestUserAgent)
		if withToken {
			// Optionally set the Authorization header if the token is present and intended for use
			req.Header.Set("Authorization", "Bearer "+opts.GitHubToken)
		}

		return req, nil
	}

	tokenApplied := false
	if opts.GitHubToken != "" {
		if opts.ForceAuthByToken {
			tokenApplied = true
		} else if parsed, err := url.Parse(opts.URL); err == nil {
			tokenApplied = shouldAuthenticate(parsed.Hostname())
		}
	}

	req, err := buildRequest(tokenApplied)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if tokenApplied && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
		// If the first authenticated requests with 401/403, assume the token is invalid/has permission issues
		if opts.OnTokenRejected != nil {
			// Callback to notify about token rejection
			opts.OnTokenRejected(resp.StatusCode)
		}
		resp.Body.Close()
    
		// Attempt again with unauthorized request
		reqNoAuth, reqErr := buildRequest(false)
		if reqErr != nil {
			return nil, reqErr
		}
		return client.Do(reqNoAuth)
	}

	return resp, nil
}
