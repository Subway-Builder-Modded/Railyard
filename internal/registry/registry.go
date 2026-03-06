package registry

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"railyard/internal/paths"
	"railyard/internal/types"
)

const RegistryRepoURL = "https://github.com/Subway-Builder-Modded/The-Railyard"

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
		repoPath: paths.RegistryRepoPath(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Initialize ensures a valid local registry repo exists.
// It does not force a remote refresh.
func (r *Registry) Initialize() error {
	if err := r.openOrClone(); err != nil {
		return err
	}

	if err := r.fetchFromDisk(); err != nil {
		return fmt.Errorf("failed to load registry data from disk: %w", err)
	}

	return nil
}

// Refresh forces a pull of the latest registry changes.
func (r *Registry) Refresh() error {
	if err := r.refreshRepo(); err != nil {
		return err
	}

	if err := r.fetchFromDisk(); err != nil {
		return fmt.Errorf("failed to load registry data from disk after refresh: %w", err)
	}
	return nil
}

// GetMods returns all mod manifests.
func (r *Registry) GetMods() []types.ModManifest {
	return r.mods
}

// GetMaps returns all map manifests.
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
