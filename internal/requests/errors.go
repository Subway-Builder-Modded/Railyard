package requests

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"

	"railyard/internal/types"
)

type APISource = types.APIErrorSource

const (
	APISourceGitHub APISource = types.APIErrorSourceGitHub
	// TODO: Add other external sources as needed
)

type APIError struct {
	Source     APISource
	StatusCode int
	Subject    string
	Cause      error
}

func (e APIError) Error() string {
	if e.StatusCode > 0 {
		base := fmt.Sprintf("%s API returned status %d for %q", e.Source, e.StatusCode, e.Subject)
		if IsAuthStatus(e.StatusCode) {
			return fmt.Sprintf(
				"%s. Add a GitHub token: %s",
				base,
				types.GitHubTokenDocsURL,
			)
		}
		return base
	}

	if e.Cause != nil {
		return fmt.Sprintf("failed to fetch %s data for %q: %v", e.Source, e.Subject, e.Cause)
	}
	return fmt.Sprintf("%s API error for %q", e.Source, e.Subject)
}

func (e APIError) Unwrap() error {
	return e.Cause
}

func IsAuthStatus(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

func classifyAPIStatus(statusCode int) types.APIErrorType {
	switch {
	case statusCode == http.StatusUnauthorized:
		return types.APIErrorTypeAuthInvalidToken
	case statusCode == http.StatusTooManyRequests:
		return types.APIErrorTypeRateLimited
	case statusCode == http.StatusForbidden:
		return types.APIErrorTypeForbidden
	case statusCode == http.StatusBadRequest:
		return types.APIErrorTypeBadRequest
	case statusCode == http.StatusNotFound:
		return types.APIErrorTypeNotFound
	case statusCode >= 500 && statusCode <= 599:
		return types.APIErrorTypeUpstream5xx
	default:
		return types.APIErrorTypeStatus
	}
}

func classifyAPICause(cause error) types.APIErrorType {
	if cause == nil {
		return ""
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return types.APIErrorTypeTimeout
	}

	var urlErr *neturl.Error
	if errors.As(cause, &urlErr) {
		if urlErr.Timeout() {
			return types.APIErrorTypeTimeout
		}
	}

	var netErr net.Error
	if errors.As(cause, &netErr) {
		if netErr.Timeout() {
			return types.APIErrorTypeTimeout
		}
		return types.APIErrorTypeNetwork
	}

	var opErr *net.OpError
	if errors.As(cause, &opErr) {
		return types.APIErrorTypeNetwork
	}

	var dnsErr *net.DNSError
	if errors.As(cause, &dnsErr) {
		return types.APIErrorTypeNetwork
	}

	return types.APIErrorTypeFetch
}

func ResolveAPIError(err error) (types.APIErrorType, types.APIErrorSource, int, bool) {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode > 0 {
			return classifyAPIStatus(apiErr.StatusCode), apiErr.Source, apiErr.StatusCode, true
		}
		if apiErr.Cause != nil {
			return classifyAPICause(apiErr.Cause), apiErr.Source, 0, true
		}
	}

	return "", "", 0, false
}
