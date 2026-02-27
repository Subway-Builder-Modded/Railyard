package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"railyard/internal/files"
	"railyard/internal/types"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

const registryRepoURL = "https://github.com/Subway-Builder-Modded/The-Railyard"

// Registry manages the local clone of The Railyard registry repository.
type Registry struct {
	repoPath      string
	httpClient    *http.Client
	mods          []types.ModManifest
	maps          []types.MapManifest
	installedMods []types.InstalledModInfo
	installedMaps []types.InstalledMapInfo
}

// NewRegistry creates a new Registry instance with the platform-appropriate
// storage path.
func NewRegistry() *Registry {
	return &Registry{
		repoPath: RegistryRepoPath(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Initialize ensures a valid local registry repo exists.
// It does not force a remote refresh.
func (r *Registry) initialize() error {
	repo, err := git.PlainOpen(r.repoPath)
	if err != nil {
		// Directory doesn't exist or isn't a valid git repo -- (re)clone.
		return r.forceClone()
	}

	// Existing repo is corrupt/unreadable; rebuild it.
	if _, err := repo.Head(); err != nil {
		return r.forceClone()
	}

	return nil
}

// Refresh forces a pull of the latest registry changes.
func (r *Registry) Refresh() error {
	repo, err := git.PlainOpen(r.repoPath)
	if err != nil {
		return r.forceClone()
	}

	if err := r.fetchAndReset(repo); err != nil {
		// If fetch/reset fails the repo may be corrupted -- delete and re-clone.
		return r.forceClone()
	}
	return nil
}

func (r *Registry) WriteInstalledToDisk() error {
	if err := files.WriteJSON[types.InstalledModFile](InstalledModsPath(), "installed mod file", r.installedMods); err != nil {
		return fmt.Errorf("failed to write installed mods to disk: %w", err)
	}
	if err := files.WriteJSON[types.InstalledMapFile](InstalledMapsPath(), "installed map file", r.installedMaps); err != nil {
		return fmt.Errorf("failed to write installed maps to disk: %w", err)
	}
	return nil
}

// fetchFromDisk loads all registry data (mods, maps, installed mods, installed maps) from disk into memory.
func (r *Registry) fetchFromDisk() error {
	var err error
	r.mods, err = r.getModsFromDisk()
	if err != nil {
		return err
	}
	r.maps, err = r.getMapsFromDisk()
	if err != nil {
		return err
	}
	r.installedMods, err = r.getInstalledModsFromDisk()
	if err != nil {
		return err
	}
	r.installedMaps, err = r.getInstalledMapsFromDisk()
	if err != nil {
		return err
	}
	return nil
}

// cloneOrUpdate handles clone-if-missing and fetch+reset-if-exists logic.
func (r *Registry) cloneOrUpdate() error {
	if err := r.ensureLocalRepo(); err != nil {
		return err
	}

	// Repo now exists and opens cleanly. Refresh to preserve legacy Initialize behavior.
	return r.Refresh()
}

// ensureLocalRepo ensures a valid local clone exists without forcing a refresh.
func (r *Registry) ensureLocalRepo() error {
	// Try to open the existing repo
	repo, err := git.PlainOpen(r.repoPath)
	if err != nil {
		// Directory doesn't exist or isn't a valid git repo -- (re)clone.
		return r.forceClone()
	}

	// Existing repo is corrupt/unreadable; rebuild it.
	if _, err := repo.Head(); err != nil {
		return r.forceClone()
	}

	if err := r.fetchFromDisk(); err != nil {
		return fmt.Errorf("failed to load registry data from disk after update: %w", err)
	}
	return nil
}

// getCredentials uses the system's git credential helper to resolve
// credentials for the registry repo URL. Returns nil auth if no
// credentials are found (for public repos).
func (r *Registry) getCredentials() *githttp.BasicAuth {
	parsed, err := url.Parse(registryRepoURL)
	if err != nil {
		return nil
	}

	input := fmt.Sprintf("protocol=%s\nhost=%s\npath=%s\n\n", parsed.Scheme, parsed.Host, strings.TrimPrefix(parsed.Path, "/"))

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil
	}

	var username, password string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "username":
				username = v
			case "password":
				password = v
			}
		}
	}

	if username != "" && password != "" {
		return &githttp.BasicAuth{
			Username: username,
			Password: password,
		}
	}
	return nil
}

// AddInstalledMod adds a mod to the in-memory list of installed mods. Remember to call WriteInstalledToDisk() to persist changes.
func (r *Registry) AddInstalledMod(modID string, version string) {
	r.installedMods = append(r.installedMods, types.InstalledModInfo{
		ID:      modID,
		Version: version,
	})
}

// AddInstalledMap adds a map to the in-memory list of installed maps. Remember to call WriteInstalledToDisk() to persist changes.
func (r *Registry) AddInstalledMap(mapID string, version string, config types.ConfigData) {
	r.installedMaps = append(r.installedMaps, types.InstalledMapInfo{
		ID:        mapID,
		Version:   version,
		MapConfig: config,
	})
}

// forceClone removes any existing directory and performs a fresh clone.
func (r *Registry) forceClone() error {
	// Remove existing directory if present
	if err := os.RemoveAll(r.repoPath); err != nil {
		return fmt.Errorf("failed to remove existing registry directory: %w", err)
	}

	// Ensure parent directory exists
	parent := filepath.Dir(r.repoPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("failed to create registry parent directory: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:           registryRepoURL,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
		Depth:         1,
	}
	if auth := r.getCredentials(); auth != nil {
		cloneOpts.Auth = auth
	}

	_, err := git.PlainClone(r.repoPath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone registry repo: %w", err)
	}
	if err := r.fetchFromDisk(); err != nil {
		return fmt.Errorf("failed to load registry data from disk after clone: %w", err)
	}

	return nil
}

// fetchAndReset fetches from origin and hard-resets the working tree to
// origin/main.
func (r *Registry) fetchAndReset(repo *git.Repository) error {
	// Fetch with force
	fetchOpts := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"+refs/heads/main:refs/remotes/origin/main",
		},
		Force: true,
	}
	if auth := r.getCredentials(); auth != nil {
		fetchOpts.Auth = auth
	}
	err := repo.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}

	// Resolve origin/main
	ref, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
	if err != nil {
		return fmt.Errorf("failed to resolve origin/main: %w", err)
	}

	// Get the worktree and hard reset
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = wt.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to origin/main: %w", err)
	}

	return nil
}

// getModsFromDisk reads the mods index and returns all mod manifests.
func (r *Registry) getModsFromDisk() ([]types.ModManifest, error) {
	indexPath := filepath.Join(r.repoPath, "mods", "index.json")
	index, err := files.ReadJSON[types.IndexFile](indexPath, "mods index", files.JSONReadOptions{})
	if err != nil {
		return nil, err
	}

	mods := make([]types.ModManifest, 0, len(index.Mods))
	for _, modID := range index.Mods {
		manifestPath := filepath.Join(r.repoPath, "mods", modID, "manifest.json")
		manifest, modErr := files.ReadJSON[types.ModManifest](manifestPath, fmt.Sprintf("manifest for mod %q", modID), files.JSONReadOptions{})
		if modErr != nil {
			return nil, modErr
		}
		mods = append(mods, manifest)
	}

	return mods, nil
}

func (r *Registry) getInstalledModsFromDisk() ([]types.InstalledModInfo, error) {
	if _, err := os.Stat(InstalledModsPath()); os.IsNotExist(err) {
		return []types.InstalledModInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledModInfo](InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
}

func (r *Registry) getInstalledMapsFromDisk() ([]types.InstalledMapInfo, error) {
	if _, err := os.Stat(InstalledMapsPath()); os.IsNotExist(err) {
		return []types.InstalledMapInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledMapInfo](InstalledMapsPath(), "installed maps file", files.JSONReadOptions{})
}

// GetMods reads the mods index and returns all mod manifests.
func (r *Registry) GetMods() []types.ModManifest {
	return r.mods
}

// GetMaps reads the maps index and returns all map manifests.
func (r *Registry) GetMaps() []types.MapManifest {
	return r.maps
}

// GetMod looks up a mod manifest by ID from the loaded registry data.
func (r *Registry) GetMod(modID string) (*types.ModManifest, error) {
	for _, m := range r.GetMods() {
		if m.ID == modID {
			return &m, nil
		}
	}

	return nil, fmt.Errorf("mod with ID %q not found in registry", modID)
}

func (d *Registry) RemoveInstalledMod(modID string) {
	updated := make([]types.InstalledModInfo, 0, len(d.installedMods))
	for _, mod := range d.installedMods {
		if mod.ID != modID {
			updated = append(updated, mod)
		}
	}
	d.installedMods = updated
}

func (d *Registry) RemoveInstalledMap(mapID string) {
	updated := make([]types.InstalledMapInfo, 0, len(d.installedMaps))
	for _, m := range d.installedMaps {
		if m.ID != mapID {
			updated = append(updated, m)
		}
	}
}

// getMapsFromDisk reads the maps index and returns all map manifests.
func (r *Registry) getMapsFromDisk() ([]types.MapManifest, error) {
	indexPath := filepath.Join(r.repoPath, "maps", "index.json")
	index, indexErr := files.ReadJSON[types.IndexFile](indexPath, "maps index", files.JSONReadOptions{})
	if indexErr != nil {
		return nil, indexErr
	}

	maps := make([]types.MapManifest, 0, len(index.Maps))
	for _, mapID := range index.Maps {
		manifestPath := filepath.Join(r.repoPath, "maps", mapID, "manifest.json")
		manifest, mapErr := files.ReadJSON[types.MapManifest](manifestPath, fmt.Sprintf("manifest for map %q", mapID), files.JSONReadOptions{})
		if mapErr != nil {
			return nil, mapErr
		}
		maps = append(maps, manifest)
	}

	return maps, nil
}

// GetMap looks up a map manifest by ID from the loaded registry data.
func (r *Registry) GetMap(mapID string) (*types.MapManifest, error) {
	for _, m := range r.GetMaps() {
		if m.ID == mapID {
			return &m, nil
		}
	}

	return nil, fmt.Errorf("map with ID %q not found in registry", mapID)
}

// GetGalleryImage reads an image file from the cloned registry repo and
// returns it as a base64 data URL suitable for use in an <img> src attribute.
//
// itemType should be "mods" or "maps".
// itemID is the mod/map identifier.
// imagePath is the relative image path as listed in the manifest's gallery field.
func (r *Registry) GetGalleryImage(itemType string, itemID string, imagePath string) (string, error) {
	// Sanitize inputs to prevent path traversal
	if strings.Contains(itemType, "..") || strings.Contains(itemID, "..") || strings.Contains(imagePath, "..") {
		return "", fmt.Errorf("invalid path component: path traversal not allowed")
	}

	fullPath := filepath.Join(r.repoPath, itemType, itemID, imagePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read gallery image %q: %w", fullPath, err)
	}

	// Detect MIME type from file extension
	mimeType := mimeFromExtension(filepath.Ext(fullPath))

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// mimeFromExtension returns the MIME type for common image file extensions.
func mimeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
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

// GetInstalledMods returns the IDs of locally installed mods.
// Currently stubbed to return an empty slice.
func (r *Registry) GetInstalledMods() []types.InstalledModInfo {
	return r.installedMods
}

// GetInstalledMaps returns the IDs of locally installed maps.
// Currently stubbed to return an empty slice.
func (r *Registry) GetInstalledMaps() []types.InstalledMapInfo {
	return r.installedMaps
}

// GetInstalledMapCodes returns the city codes of locally installed maps.
// Currently stubbed to return an empty slice.
func (r *Registry) GetInstalledMapCodes() []string {
	codes := make([]string, 0, len(r.installedMaps))
	for _, m := range r.installedMaps {
		codes = append(codes, m.MapConfig.Code)
	}
	return codes
}
