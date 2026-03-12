package registry

import (
	"fmt"
	"path/filepath"

	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/types"
	"railyard/internal/utils"
)

// fetchFromDisk loads all registry data (mods, maps, installed mods, installed maps) from disk into memory.
func (r *Registry) fetchFromDisk() error {
	mods, err := r.getModsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to load mods from disk: %w", err)
	}

	maps, err := r.getMapsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to load maps from disk: %w", err)
	}

	downloadCounts, err := r.loadDownloadCounts([]types.AssetType{
		types.AssetTypeMap,
		types.AssetTypeMod,
	})
	if err != nil {
		return err
	}

	installedMods, err := r.getInstalledModsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to load installed mods from disk: %w", err)
	}

	installedMaps, err := r.getInstalledMapsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to load installed maps from disk: %w", err)
	}

	modLastUpdated, mapLastUpdated := r.loadLastUpdated(mods, maps)
	updateManifestLastUpdated(mods, maps, modLastUpdated, mapLastUpdated)

	// Make updates only when all reads are successful to avoid partial registry updates
	r.mods = mods
	r.maps = maps
	r.downloadCounts = downloadCounts
	r.installedMods = installedMods
	r.installedMaps = installedMaps

	return nil
}

// getModsFromDisk reads the mods index and returns all mod manifests.
func (r *Registry) getModsFromDisk() ([]types.ModManifest, error) {
	indexPath := filepath.Join(r.repoPath, "mods", constants.INDEX_JSON)
	index, err := files.ReadJSON[types.IndexFile](indexPath, "mods index", files.JSONReadOptions{})
	if err != nil {
		return nil, err
	}
	return readManifestsFromDisk[types.ModManifest](r.repoPath, "mods", "mod", index.Mods)
}

func (r *Registry) SetInstalledMapsFromPath(path string) error {
	installedMaps, err := files.ReadJSON[[]types.InstalledMapInfo](path, "installed maps file", files.JSONReadOptions{})
	if err != nil {
		return fmt.Errorf("failed to read installed maps from path %q: %w", path, err)
	}
	r.installedMaps = installedMaps
	return nil
}

func (r *Registry) SetInstalledModsFromPath(path string) error {
	installedMods, err := files.ReadJSON[[]types.InstalledModInfo](path, "installed mods file", files.JSONReadOptions{})
	if err != nil {
		return fmt.Errorf("failed to read installed mods from path %q: %w", path, err)
	}
	r.installedMods = installedMods
	return nil
}

// getMapsFromDisk reads the maps index and returns all map manifests.
func (r *Registry) getMapsFromDisk() ([]types.MapManifest, error) {
	indexPath := filepath.Join(r.repoPath, "maps", constants.INDEX_JSON)
	index, indexErr := files.ReadJSON[types.IndexFile](indexPath, "maps index", files.JSONReadOptions{})
	if indexErr != nil {
		return nil, indexErr
	}
	return readManifestsFromDisk[types.MapManifest](r.repoPath, "maps", "map", index.Maps)
}

func (r *Registry) getDownloadCountsFromDisk(assetType types.AssetType) (map[string]map[string]int, error) {
	assetDir := types.AssetTypeDir(assetType)
	downloadsPath := filepath.Join(r.repoPath, assetDir, constants.DOWNLOADS_JSON)
	downloadsFile, err := files.ReadJSON[types.DownloadsFile](downloadsPath, fmt.Sprintf("%s download counts", assetType), files.JSONReadOptions{})
	if err != nil {
		return nil, err
	}
	return utils.CloneNestedMap(utils.OrEmptyNestedMap(downloadsFile)), nil
}

func (r *Registry) loadDownloadCounts(assetTypes []types.AssetType) (map[types.AssetType]map[string]map[string]int, error) {
	countsByType := make(map[types.AssetType]map[string]map[string]int, len(assetTypes))
	for _, assetType := range assetTypes {
		counts, err := r.getDownloadCountsFromDisk(assetType)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s download counts from disk: %w", assetType, err)
		}
		countsByType[assetType] = counts
	}
	return countsByType, nil
}

func readManifestsFromDisk[T any](repoPath string, assetDir string, assetLabel string, ids []string) ([]T, error) {
	manifests := make([]T, 0, len(ids))
	for _, assetID := range ids {
		manifestPath := filepath.Join(repoPath, assetDir, assetID, constants.MANIFEST_JSON)
		manifest, err := files.ReadJSON[T](manifestPath, fmt.Sprintf("manifest for %s %q", assetLabel, assetID), files.JSONReadOptions{})
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}
