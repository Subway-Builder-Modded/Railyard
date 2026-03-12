package types

// UpdateConfig describes how a mod or map receives updates.
type UpdateConfig struct {
	Type string `json:"type"`
	Repo string `json:"repo,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Source returns the canonical source identifier for update resolution.
// For GitHub updates this is the repo slug; otherwise it is the direct URL.
func (u UpdateConfig) Source() string {
	if u.Type == "github" {
		return u.Repo
	}
	return u.URL
}

// ModManifest is the manifest schema for a mod entry in the registry.
type ModManifest struct {
	SchemaVersion int          `json:"schema_version"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Author        string       `json:"author"`
	GithubID      int          `json:"github_id"`
	LastUpdated   int64        `json:"last_updated"`
	Description   string       `json:"description"`
	Tags          []string     `json:"tags"`
	Gallery       []string     `json:"gallery"`
	Source        string       `json:"source"`
	Update        UpdateConfig `json:"update"`
}

// InstalledModInfo represents the information stored about an installed mod in the registry's installed_mods.json file.
type InstalledModInfo struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// InstalledMapInfo represents the information stored about an installed map in the registry's installed_maps.json file.
type InstalledMapInfo struct {
	ID        string     `json:"id"`
	Version   string     `json:"version"`
	MapConfig ConfigData `json:"config"`
}

// InstalledModFile represents the structure of the installed_mods.json file, which is a list of installed mods.
type InstalledModFile []InstalledModInfo

// InstalledMapFile represents the structure of the installed_maps.json file, which is a list of installed maps.
type InstalledMapFile []InstalledMapInfo

// MapManifest is the manifest schema for a map entry in the registry.
type MapManifest struct {
	SchemaVersion int          `json:"schema_version"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Author        string       `json:"author"`
	GithubID      int          `json:"github_id"`
	LastUpdated   int64        `json:"last_updated"`
	CityCode      string       `json:"city_code"`
	Country       string       `json:"country"`
	Location      string       `json:"location"`
	Population    int          `json:"population"`
	Description   string       `json:"description"`
	DataSource    string       `json:"data_source"`
	SourceQuality string       `json:"source_quality"`
	LevelOfDetail string       `json:"level_of_detail"`
	SpecialDemand []string     `json:"special_demand"`
	Tags          []string     `json:"tags"`
	Gallery       []string     `json:"gallery"`
	Source        string       `json:"source"`
	Update        UpdateConfig `json:"update"`
}

// IndexFile represents the top-level index.json in the mods/ or maps/ directory.
type IndexFile struct {
	SchemaVersion int      `json:"schema_version"`
	Mods          []string `json:"mods,omitempty"`
	Maps          []string `json:"maps,omitempty"`
}

// DownloadsFile represents downloads.json on disk, keyed by asset ID then version.
// Example:
//
//	{
//	  "calgary": { "1.1.3": 60, "1.0.1": 62 },
//	  "dublin": { "v1.0.0": 76 }
//	}
type DownloadsFile map[string]map[string]int

type AssetDownloadCountsResponse struct {
	GenericResponse
	AssetType string         `json:"assetType"`
	AssetID   string         `json:"assetId"`
	Counts    map[string]int `json:"counts"`
}

type DownloadCountsByAssetTypeResponse struct {
	GenericResponse
	AssetType string                    `json:"assetType"`
	Counts    map[string]map[string]int `json:"counts"`
}

// VersionInfo represents a single release version for a mod or map.
type VersionInfo struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Changelog   string `json:"changelog"`
	Date        string `json:"date"`
	DownloadURL string `json:"download_url"`
	GameVersion string `json:"game_version"`
	SHA256      string `json:"sha256"`
	Downloads   int    `json:"downloads"`
	Manifest    string `json:"manifest,omitempty"`
	Prerelease  bool   `json:"prerelease"`
}

// GithubRelease maps fields from the GitHub Releases API response.
type GithubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	Assets      []GithubAsset `json:"assets"`
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	DownloadCount      int    `json:"download_count"`
}

// CustomUpdateFile maps the custom update.json schema.
type CustomUpdateFile struct {
	SchemaVersion int                   `json:"schema_version"`
	Versions      []CustomUpdateVersion `json:"versions"`
}

type CustomUpdateVersion struct {
	Version     string `json:"version"`
	GameVersion string `json:"game_version"`
	Date        string `json:"date"`
	Changelog   string `json:"changelog"`
	Download    string `json:"download"`
	SHA256      string `json:"sha256"`
	Manifest    string `json:"manifest,omitempty"`
}
