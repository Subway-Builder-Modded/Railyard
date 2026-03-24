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
	"railyard/internal/requests"
	"railyard/internal/types"
)

// modManifestDeps is the minimal schema needed to extract dependencies from a mod's manifest.json.
type modManifestDeps struct {
	Dependencies map[string]string `json:"dependencies"`
}

var registryGitHubAPIBaseURL = types.GitHubAPIBaseURL

// GetVersions fetches available versions for a mod or map.
// updateType must be "github" or "custom".
// repoOrURL is "owner/repo" for github, or a URL for custom.
func (r *Registry) GetVersions(updateType string, repoOrURL string) ([]types.VersionInfo, error) {
	// Check cache first to avoid redundant network requests
	cacheKey := updateType + "|" + repoOrURL
	if cached, ok := r.getCachedVersions(cacheKey); ok {
		return cached, nil
	}

	var (
		versions []types.VersionInfo
		err      error
	)

	switch updateType {
	case "github":
		versions, err = r.getGitHubVersions(repoOrURL)
	case "custom":
		versions, err = r.getCustomVersions(repoOrURL)
	default:
		return nil, fmt.Errorf("unsupported update type: %q", updateType)
	}

	if err != nil {
		return nil, err
	}

	// If version resolution succeeded, cache the results for future calls
	r.setCachedVersions(cacheKey, versions)
	// Return a copy of the versions to prevent external mutation by callers
	return cloneVersionInfos(versions), nil
}

// GetVersionsResponse fetches available versions and reports status metadata.
func (r *Registry) GetVersionsResponse(updateType string, repoOrURL string) types.VersionsResponse {
	versions, err := r.GetVersions(updateType, repoOrURL)
	if err != nil {
		return types.VersionsResponse{
			GenericResponse: types.ErrorResponse(err.Error()),
			Versions:        []types.VersionInfo{},
		}
	}

	return types.VersionsResponse{
		GenericResponse: types.SuccessResponse("Versions loaded"),
		Versions:        versions,
	}
}

func (r *Registry) getGitHubVersions(repo string) ([]types.VersionInfo, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid GitHub repo format %q: expected \"owner/repo\"", repo)
	}

	baseURL := strings.TrimRight(registryGitHubAPIBaseURL, "/")
	apiURL := fmt.Sprintf("%s/repos/%s/releases", baseURL, repo)

	resp, err := requests.GetWithGithubToken(r.httpClient, requests.GithubTokenRequestArgs{
		URL:              apiURL,
		GitHubToken:      r.config.GetGithubToken(),
		ForceAuthByToken: true,
		Headers: map[string]string{
			"Accept": "application/vnd.github+json",
		},
		OnTokenRejected: func(statusCode int) {
			r.logger.Warn("GitHub token rejected; retrying unauthenticated request", "repo", repo, "status", statusCode)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub releases for %q: %w", repo, err)
	}

	if resp.StatusCode != http.StatusOK {
		status := resp.StatusCode
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API returned status %d for %q", status, repo)
	}
	defer resp.Body.Close()

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

	versions = r.filterSemverVersions(versions, "github:"+repo)
	// Fetch manifest.json assets in parallel to extract game_version
	r.enrichVersions(versions)

	return versions, nil
}

// enrichVersions fetches manifest.json URLs in parallel and populates GameVersion and dependencies
// from the game dependency key in the manifest. Errors are silently ignored per-version.
func (r *Registry) enrichVersions(versions []types.VersionInfo) {
	var wg sync.WaitGroup
	for i := range versions {
		if versions[i].Manifest == "" {
			continue
		}
		wg.Add(1)
		go func(v *types.VersionInfo) {
			defer wg.Done()
			resp, err := requests.GetWithGithubToken(r.httpClient, requests.GithubTokenRequestArgs{
				URL: v.Manifest,
				Headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			})
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
			newDeps := make(map[string]string)
			for depID, depRange := range manifest.Dependencies {
				if depID == constants.GameDependencyKey {
					continue
				}
				newDeps[depID] = depRange
			}
			v.Dependencies = newDeps
		}(&versions[i])
	}
	wg.Wait()
}

func (r *Registry) getCustomVersions(updateURL string) ([]types.VersionInfo, error) {
	parsed, err := url.Parse(updateURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid custom update URL %q: must be http or https", updateURL)
	}

	resp, err := requests.GetWithGithubToken(r.httpClient, requests.GithubTokenRequestArgs{
		URL: updateURL,
	})
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

	return r.filterSemverVersions(versions, "custom:"+updateURL), nil
}

func (r *Registry) filterSemverVersions(
	versions []types.VersionInfo,
	sourceLabel string,
) []types.VersionInfo {
	filtered := make([]types.VersionInfo, 0, len(versions))
	for _, version := range versions {
		if !types.IsValidSemverVersion(types.Version(version.Version)) {
			r.logger.Warn("Skipping non-semver version", "version", version.Version, "source", sourceLabel)
			continue
		}
		filtered = append(filtered, version)
	}
	return filtered
}

func cloneVersionInfos(input []types.VersionInfo) []types.VersionInfo {
	output := make([]types.VersionInfo, len(input))
	copy(output, input)
	return output
}

func (r *Registry) getCachedVersions(key string) ([]types.VersionInfo, bool) {
	r.versionsMu.RLock()
	defer r.versionsMu.RUnlock()
	versions, ok := r.versionsCache[key]
	if !ok {
		return nil, false
	}
	return cloneVersionInfos(versions), true
}

func (r *Registry) setCachedVersions(key string, versions []types.VersionInfo) {
	r.versionsMu.Lock()
	defer r.versionsMu.Unlock()
	r.versionsCache[key] = cloneVersionInfos(versions)
}

func (r *Registry) clearVersionsCache() {
	r.versionsMu.Lock()
	defer r.versionsMu.Unlock()
	r.versionsCache = map[string][]types.VersionInfo{}
}
