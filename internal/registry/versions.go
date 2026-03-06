package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"railyard/internal/types"
)

// GetVersions fetches available versions for a mod or map.
// updateType must be "github" or "custom".
// repoOrURL is "owner/repo" for github, or a URL for custom.
func (r *Registry) GetVersions(updateType string, repoOrURL string) ([]types.VersionInfo, error) {
	switch updateType {
	case "github":
		return r.getGitHubVersions(repoOrURL)
	case "custom":
		return r.getCustomVersions(repoOrURL)
	default:
		return nil, fmt.Errorf("unsupported update type: %q", updateType)
	}
}

func (r *Registry) getGitHubVersions(repo string) ([]types.VersionInfo, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid GitHub repo format %q: expected \"owner/repo\"", repo)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Railyard-Desktop-App")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub releases for %q: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for %q", resp.StatusCode, repo)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	var releases []types.GithubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub releases JSON: %w", err)
	}

	versions := make([]types.VersionInfo, 0, len(releases))
	for _, rel := range releases {
		v := types.VersionInfo{
			Version:   rel.TagName,
			Name:      rel.Name,
			Changelog: rel.Body,
			Date:      rel.PublishedAt,
		}
		for _, asset := range rel.Assets {
			v.Downloads += asset.DownloadCount
		}
		if len(rel.Assets) > 0 {
			v.DownloadURL = rel.Assets[0].BrowserDownloadURL
		}
		versions = append(versions, v)
	}

	return versions, nil
}

func (r *Registry) getCustomVersions(updateURL string) ([]types.VersionInfo, error) {
	parsed, err := url.Parse(updateURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid custom update URL %q: must be http or https", updateURL)
	}

	req, err := http.NewRequest("GET", updateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for custom update URL: %w", err)
	}
	req.Header.Set("User-Agent", "Railyard-Desktop-App")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch custom update from %q: %w", updateURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("custom update URL returned status %d for %q", resp.StatusCode, updateURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read custom update response: %w", err)
	}

	var updateFile types.CustomUpdateFile
	if err := json.Unmarshal(body, &updateFile); err != nil {
		return nil, fmt.Errorf("failed to parse custom update JSON: %w", err)
	}

	versions := make([]types.VersionInfo, 0, len(updateFile.Versions))
	for _, v := range updateFile.Versions {
		versions = append(versions, types.VersionInfo{
			Version:     v.Version,
			Name:        v.Version,
			Changelog:   v.Changelog,
			Date:        v.Date,
			DownloadURL: v.Download,
			GameVersion: v.GameVersion,
			SHA256:      v.SHA256,
		})
	}

	return versions, nil
}
