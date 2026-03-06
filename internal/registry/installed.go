package registry

import (
	"fmt"
	"os"

	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/types"
)

// WriteInstalledToDisk persists installed mods and maps state to disk.
func (r *Registry) WriteInstalledToDisk() error {
	if err := files.WriteJSON[types.InstalledModFile](paths.InstalledModsPath(), "installed mod file", r.installedMods); err != nil {
		return fmt.Errorf("failed to write installed mods to disk: %w", err)
	}
	if err := files.WriteJSON[types.InstalledMapFile](paths.InstalledMapsPath(), "installed map file", r.installedMaps); err != nil {
		return fmt.Errorf("failed to write installed maps to disk: %w", err)
	}
	return nil
}

func (r *Registry) getInstalledModsFromDisk() ([]types.InstalledModInfo, error) {
	if _, err := os.Stat(paths.InstalledModsPath()); os.IsNotExist(err) {
		return []types.InstalledModInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledModInfo](paths.InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
}

func (r *Registry) getInstalledMapsFromDisk() ([]types.InstalledMapInfo, error) {
	if _, err := os.Stat(paths.InstalledMapsPath()); os.IsNotExist(err) {
		return []types.InstalledMapInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledMapInfo](paths.InstalledMapsPath(), "installed maps file", files.JSONReadOptions{})
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

func (r *Registry) RemoveInstalledMod(modID string) {
	updated := make([]types.InstalledModInfo, 0, len(r.installedMods))
	for _, mod := range r.installedMods {
		if mod.ID != modID {
			updated = append(updated, mod)
		}
	}
	r.installedMods = updated
}

func (r *Registry) RemoveInstalledMap(mapID string) {
	updated := make([]types.InstalledMapInfo, 0, len(r.installedMaps))
	for _, m := range r.installedMaps {
		if m.ID != mapID {
			updated = append(updated, m)
		}
	}
	r.installedMaps = updated
}

// GetInstalledMods returns the locally installed mods.
func (r *Registry) GetInstalledMods() []types.InstalledModInfo {
	return r.installedMods
}

// GetInstalledMaps returns the locally installed maps.
func (r *Registry) GetInstalledMaps() []types.InstalledMapInfo {
	return r.installedMaps
}

// GetInstalledMapCodes returns the city codes of locally installed maps.
func (r *Registry) GetInstalledMapCodes() []string {
	codes := make([]string, 0, len(r.installedMaps))
	for _, m := range r.installedMaps {
		codes = append(codes, m.MapConfig.Code)
	}
	return codes
}
