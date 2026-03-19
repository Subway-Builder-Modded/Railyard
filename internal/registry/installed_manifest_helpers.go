package registry

import (
	"railyard/internal/types"
)

func mapConfigFromManifest(manifest *types.MapManifest, version string) types.ConfigData {
	config := types.ConfigData{}
	if manifest == nil {
		config.Version = version
		return config
	}

	config.Code = manifest.CityCode
	config.Name = manifest.Name
	config.Description = manifest.Description
	config.Population = manifest.Population
	config.Creator = manifest.Author
	config.Version = version
	config.Country = &manifest.Country
	config.InitialViewState = manifest.InitialViewState

	return config
}

func installedMapInfoFromManifest(mapID string, version string, manifest *types.MapManifest) types.InstalledMapInfo {
	return types.InstalledMapInfo{
		ID:        mapID,
		Version:   version,
		MapConfig: mapConfigFromManifest(manifest, version),
	}
}

func installedModInfoFromManifest(modID string, version string, manifest *types.ModManifest) types.InstalledModInfo {
	if manifest != nil && modID == "" {
		modID = manifest.ID
	}
	return types.InstalledModInfo{
		ID:      modID,
		Version: version,
	}
}
