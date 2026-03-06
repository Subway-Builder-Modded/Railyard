package registry

import (
	"fmt"
	"path/filepath"

	"railyard/internal/files"
	"railyard/internal/types"
)

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
