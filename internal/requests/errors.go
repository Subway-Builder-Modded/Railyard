package requests

import (
	"fmt"
	"net/http"

	"railyard/internal/types"
)

type APISource string

const (
	APISourceGitHub APISource = "Github"
	// TODO: Add other external sources as needed
)

type APIStatusError struct {
	Source     APISource
	StatusCode int
	Subject    string
}

func (e APIStatusError) Error() string {
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

func NewAPIStatusError(source APISource, statusCode int, subject string) error {
	return APIStatusError{
		Source:     source,
		StatusCode: statusCode,
		Subject:    subject,
	}
}

type APIFetchError struct {
	Source  APISource
	Subject string
	Cause   error
}

func (e APIFetchError) Error() string {
	return fmt.Sprintf("failed to fetch %s data for %q: %v", e.Source, e.Subject, e.Cause)
}

func (e APIFetchError) Unwrap() error {
	return e.Cause
}

func NewAPIFetchError(source APISource, subject string, cause error) error {
	return APIFetchError{
		Source:  source,
		Subject: subject,
		Cause:   cause,
	}
}

func IsAuthStatus(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}
