package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/types"
)

// WriteInstalledToDisk persists installed mods and maps state to disk.
func (r *Registry) WriteInstalledToDisk() error {
	// Indent JSON for readability
	modsJSON, err := json.MarshalIndent(types.InstalledModFile(r.installedMods), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize installed mod file: %w", err)
	}
	mapsJSON, err := json.MarshalIndent(types.InstalledMapFile(r.installedMaps), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize installed map file: %w", err)
	}

	if err := files.WriteFilesAtomically([]files.AtomicFileWrite{
		{
			Path:  paths.InstalledModsPath(),
			Label: "installed mod file",
			Data:  modsJSON,
			Perm:  0o644,
		},
		{
			Path:  paths.InstalledMapsPath(),
			Label: "installed map file",
			Data:  mapsJSON,
			Perm:  0o644,
		},
	}); err != nil {
		return fmt.Errorf("failed to write installed state transactionally: %w", err)
	}

	return nil
}

func (r *Registry) getInstalledModsFromDisk() ([]types.InstalledModInfo, error) {
	if _, err := os.Stat(paths.InstalledModsPath()); errors.Is(err, fs.ErrNotExist) {
		return []types.InstalledModInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledModInfo](paths.InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
}

func (r *Registry) getInstalledMapsFromDisk() ([]types.InstalledMapInfo, error) {
	if _, err := os.Stat(paths.InstalledMapsPath()); errors.Is(err, fs.ErrNotExist) {
		return []types.InstalledMapInfo{}, nil
	}

	return files.ReadJSON[[]types.InstalledMapInfo](paths.InstalledMapsPath(), "installed maps file", files.JSONReadOptions{})
}

// AddInstalledMod adds a mod to the in-memory list of installed mods. Remember to call WriteInstalledToDisk() to persist changes.
func (r *Registry) AddInstalledMod(modID string, version string, isLocal bool) {
	r.installedMods = append(r.installedMods, types.InstalledModInfo{
		ID:      modID,
		Version: version,
		IsLocal: isLocal,
	})
}

// AddInstalledMap adds a map to the in-memory list of installed maps. Remember to call WriteInstalledToDisk() to persist changes.
func (r *Registry) AddInstalledMap(mapID string, version string, isLocal bool, config types.ConfigData) {
	r.installedMaps = append(r.installedMaps, types.InstalledMapInfo{
		ID:        mapID,
		Version:   version,
		IsLocal:   isLocal,
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

// GetInstalledModsResponse returns installed mods with status metadata.
func (r *Registry) GetInstalledModsResponse() types.InstalledModsResponse {
	return types.InstalledModsResponse{
		GenericResponse: types.SuccessResponse("Installed mods loaded"),
		Mods:            r.installedMods,
	}
}

// GetInstalledMaps returns the locally installed maps.
func (r *Registry) GetInstalledMaps() []types.InstalledMapInfo {
	return r.installedMaps
}

// GetRemoteInstalledMaps returns installed maps excluding local imports.
func (r *Registry) GetRemoteInstalledMaps() []types.InstalledMapInfo {
	remote := make([]types.InstalledMapInfo, 0, len(r.installedMaps))
	for _, item := range r.installedMaps {
		if item.IsLocal {
			continue
		}
		remote = append(remote, item)
	}
	return remote
}

// GetInstalledMapsResponse returns installed maps with status metadata.
func (r *Registry) GetInstalledMapsResponse() types.InstalledMapsResponse {
	return types.InstalledMapsResponse{
		GenericResponse: types.SuccessResponse("Installed maps loaded"),
		Maps:            r.installedMaps,
	}
}

// GetInstalledMapCodes returns the city codes of locally installed maps.
func (r *Registry) GetInstalledMapCodes() []string {
	codes := make([]string, 0, len(r.installedMaps))
	for _, m := range r.installedMaps {
		codes = append(codes, m.MapConfig.Code)
	}
	return codes
}
