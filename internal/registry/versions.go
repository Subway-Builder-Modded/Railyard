package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"railyard/internal/constants"
	"railyard/internal/types"
)

// modManifestDeps is the minimal schema needed to extract dependencies from a mod's manifest.json.
type modManifestDeps struct {
	Dependencies map[string]string `json:"dependencies"`
}

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
			Version:    rel.TagName,
			Name:       rel.Name,
			Changelog:  rel.Body,
			Date:       rel.PublishedAt,
			Prerelease: rel.Prerelease,
		}
		for _, asset := range rel.Assets {
			v.Downloads += asset.DownloadCount
			if asset.Name == constants.MANIFEST_JSON {
				v.Manifest = asset.BrowserDownloadURL
			}
			if v.DownloadURL == "" && path.Ext(asset.Name) == ".zip" {
				v.DownloadURL = asset.BrowserDownloadURL
			}
		}
		versions = append(versions, v)
	}

	// Fetch manifest.json assets in parallel to extract game_version
	r.enrichGameVersions(versions)

	return versions, nil
}

// enrichGameVersions fetches manifest.json URLs in parallel and populates GameVersion
// from the game dependency key in the manifest. Errors are silently ignored per-version.
func (r *Registry) enrichGameVersions(versions []types.VersionInfo) {
	var wg sync.WaitGroup
	for i := range versions {
		if versions[i].Manifest == "" {
			continue
		}
		wg.Add(1)
		go func(v *types.VersionInfo) {
			defer wg.Done()
			req, err := http.NewRequest("GET", v.Manifest, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "Railyard-Desktop-App")
			req.Header.Set("Accept", "application/octet-stream")
			resp, err := r.httpClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return
			}
			body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
			if err != nil {
				return
			}
			var manifest modManifestDeps
			if err := json.Unmarshal(body, &manifest); err != nil {
				return
			}
			if sbRange, ok := manifest.Dependencies[constants.GameDependencyKey]; ok {
				v.GameVersion = sbRange
			}
		}(&versions[i])
	}
	wg.Wait()
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
			Manifest:    v.Manifest,
		})
	}

	return versions, nil
}
