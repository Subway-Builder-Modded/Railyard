package registry

import (
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/config"
	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/testutil"
	"railyard/internal/testutil/registrytest"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func writeInstalledMapFiles(t *testing.T, mapInstallRoot string, tilesRoot string, code string, config types.ConfigData) {
	t.Helper()
	mapDir := paths.JoinLocalPath(mapInstallRoot, code)
	require.NoError(t, os.MkdirAll(mapDir, 0o755))
	require.NoError(t, os.MkdirAll(tilesRoot, 0o755))

	require.NoError(t, files.WriteJSON(paths.JoinLocalPath(mapDir, "config.json"), "installed map config", config))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapDir, "demand_data.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapDir, "roads.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapDir, "runways_taxiways.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapDir, "buildings_index.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(tilesRoot, code+".pmtiles"), []byte("tiles"), 0o644))
}

func TestWriteInstalledToDiskPersistsMapsAndMods(t *testing.T) {
	testutil.NewHarness(t)
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	reg.installedMods = []types.InstalledModInfo{
		{ID: "mod-a", Version: "1.0.0"},
	}
	reg.installedMaps = []types.InstalledMapInfo{
		{ID: "map-a", Version: "1.0.0", MapConfig: types.ConfigData{Code: "AAA"}},
	}

	require.NoError(t, reg.WriteInstalledToDisk())

	mods, modErr := files.ReadJSON[[]types.InstalledModInfo](paths.InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
	require.NoError(t, modErr)
	require.Equal(t, reg.installedMods, mods)

	maps, mapErr := files.ReadJSON[[]types.InstalledMapInfo](paths.InstalledMapsPath(), "installed maps file", files.JSONReadOptions{})
	require.NoError(t, mapErr)
	require.Equal(t, reg.installedMaps, maps)
}

func TestWriteInstalledToDiskRollsBackWhenOnePathFails(t *testing.T) {
	testutil.NewHarness(t)
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))

	originalMods := []types.InstalledModInfo{
		{ID: "mod-old", Version: "0.9.0"},
	}
	require.NoError(t, files.WriteJSON(paths.InstalledModsPath(), "installed mods file", originalMods))

	require.NoError(t, os.RemoveAll(paths.InstalledMapsPath()))
	require.NoError(t, os.MkdirAll(paths.InstalledMapsPath(), 0o755))

	reg.installedMods = []types.InstalledModInfo{
		{ID: "mod-new", Version: "1.0.0"},
	}
	reg.installedMaps = []types.InstalledMapInfo{
		{ID: "map-new", Version: "1.0.0", MapConfig: types.ConfigData{Code: "NEW"}},
	}

	err := reg.WriteInstalledToDisk()
	require.Error(t, err)

	persistedMods, readErr := files.ReadJSON[[]types.InstalledModInfo](paths.InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
	require.NoError(t, readErr)
	require.Equal(t, originalMods, persistedMods)
}

func TestFetchFromDiskRecoversFromCorruptedInstalledState(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Mods: []types.ModManifest{
			{ID: "mod-a"},
		},
		Maps: []types.MapManifest{
			{ID: "map-a", CityCode: "AAA"},
		},
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(paths.InstalledModsPath()), 0o755))
	require.NoError(t, os.WriteFile(paths.InstalledModsPath(), []byte("{invalid"), 0o644))
	require.NoError(t, os.WriteFile(paths.InstalledMapsPath(), []byte("{invalid"), 0o644))

	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))
	require.NoError(t, reg.fetchFromDisk())
	require.Empty(t, reg.GetInstalledMods())
	require.Empty(t, reg.GetInstalledMaps())
}

func TestBootstrapInstalledStateFromProfileSkipsModOnVersionMismatch(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Mods: []types.ModManifest{
			{ID: "mod-a"},
		},
	})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	modPath := paths.JoinLocalPath(cfg.Cfg.MetroMakerDataPath, "mods", "mod-a")
	require.NoError(t, os.MkdirAll(modPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(modPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	require.NoError(t, files.WriteJSON(
		paths.JoinLocalPath(modPath, constants.MANIFEST_JSON),
		"installed mod manifest",
		types.MetroMakerModManifest{Version: "2.0.0"},
	))

	profile := types.DefaultProfile()
	profile.Subscriptions.Mods["mod-a"] = "1.0.0" // Version mismatch with manifest

	err := reg.BootstrapInstalledStateFromProfile(profile)
	require.NoError(t, err)
	require.Empty(t, reg.GetInstalledMods())
}

func TestBootstrapInstalledStateFromProfileSkipsMissingRequiredData(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Maps: []types.MapManifest{
			{ID: "map-a", CityCode: "AAA"},
			{ID: "map-empty", CityCode: ""}, // No city code
		},
	})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	profile := types.DefaultProfile()
	profile.Subscriptions.Maps["map-a"] = "1.0.0"       // Missing marker
	profile.Subscriptions.Maps["map-empty"] = "1.0.0"   // Missing city code
	profile.Subscriptions.Maps["map-missing"] = "1.0.0" // Missing manifest

	err := reg.BootstrapInstalledStateFromProfile(profile)
	require.NoError(t, err)
	require.Empty(t, reg.GetInstalledMods())
	require.Empty(t, reg.GetInstalledMaps())
}

func TestBootstrapInstalledStateFromProfileSuccessOnEmptyState(t *testing.T) {
	testutil.NewHarness(t)
	country := "IT"
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Mods: []types.ModManifest{
			{ID: "mod-a", Name: "Mod A"},
		},
		Maps: []types.MapManifest{
			{
				ID:          "map-a",
				CityCode:    "AAA",
				Name:        "Map A",
				Description: "Map Description",
				Author:      "Author A",
				Country:     country,
				Population:  123456,
				InitialViewState: struct {
					Latitude  float64  `json:"latitude"`
					Longitude float64  `json:"longitude"`
					Zoom      float64  `json:"zoom"`
					Pitch     *float64 `json:"pitch,omitempty"`
					Bearing   float64  `json:"bearing"`
				}{
					Latitude:  40.8518,
					Longitude: 14.2681,
					Zoom:      13,
					Bearing:   0,
				},
			},
		},
	})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	modPath := paths.JoinLocalPath(cfg.Cfg.MetroMakerDataPath, "mods", "mod-a")
	require.NoError(t, os.MkdirAll(modPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(modPath, constants.RailyardAssetMarker), []byte(""), 0o644)) // Add asset marker
	require.NoError(t, files.WriteJSON(
		paths.JoinLocalPath(modPath, constants.MANIFEST_JSON),
		"installed mod manifest",
		types.MetroMakerModManifest{Version: "1.0.0"},
	))

	mapPath := paths.JoinLocalPath(cfg.Cfg.MetroMakerDataPath, "cities", "data", "AAA")
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644)) // Add asset marker
	writeInstalledMapFiles(t, cfg.Cfg.GetMapsFolderPath(), paths.TilesPath(), "AAA", types.ConfigData{
		Code:        "AAA",
		Name:        "Map A",
		Description: "Map Description",
		Population:  123456,
		Creator:     "Author A",
		Country:     &country,
		Version:     "2.0.0",
	})

	profile := types.DefaultProfile()
	profile.Subscriptions.Mods["mod-a"] = "1.0.0"
	profile.Subscriptions.Maps["map-a"] = "2.0.0"

	err := reg.BootstrapInstalledStateFromProfile(profile)
	require.NoError(t, err)

	// All markers / manifests are present + valid so subscriptions and installed state should be in sync
	require.Equal(t, []types.InstalledModInfo{
		{ID: "mod-a", Version: "1.0.0"},
	}, reg.GetInstalledMods())
	require.Equal(t, []types.InstalledMapInfo{
		{
			ID:      "map-a",
			Version: "2.0.0",
			MapConfig: types.ConfigData{
				Code:        "AAA",
				Name:        "Map A",
				Description: "Map Description",
				Population:  123456,
				Creator:     "Author A",
				Country:     &country,
				Version:     "2.0.0",
				InitialViewState: struct {
					Latitude  float64  `json:"latitude"`
					Longitude float64  `json:"longitude"`
					Zoom      float64  `json:"zoom"`
					Pitch     *float64 `json:"pitch,omitempty"`
					Bearing   float64  `json:"bearing"`
				}{
					Latitude:  40.8518,
					Longitude: 14.2681,
					Zoom:      13,
					Bearing:   0,
				},
			},
		},
	}, reg.GetInstalledMaps())

	// Validate that the recovered installed state is persisted to disk
	modsOnDisk, err := files.ReadJSON[[]types.InstalledModInfo](paths.InstalledModsPath(), "installed mods file", files.JSONReadOptions{})
	require.NoError(t, err)
	require.Equal(t, reg.GetInstalledMods(), modsOnDisk)

	mapsOnDisk, err := files.ReadJSON[[]types.InstalledMapInfo](paths.InstalledMapsPath(), "installed maps file", files.JSONReadOptions{})
	require.NoError(t, err)
	require.Equal(t, reg.GetInstalledMaps(), mapsOnDisk)
}

func TestBootstrapInstalledStateFromProfilePreservesExistingRemoteMapWhenManifestMissing(t *testing.T) {
	testutil.NewHarness(t)
	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	reg.installedMaps = []types.InstalledMapInfo{
		{
			ID:      "missing-map",
			Version: "1.0.0",
			IsLocal: false,
			MapConfig: types.ConfigData{
				Code:    "MIS",
				Name:    "Missing",
				Version: "1.0.0",
				Country: func() *string { value := "US"; return &value }(),
			},
		},
	}

	require.NoError(t, os.MkdirAll(paths.RegistryRepoPath(), 0o755))
	reg.maps = []types.MapManifest{}
	mapPath := paths.JoinLocalPath(cfg.Cfg.GetMapsFolderPath(), "MIS")
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	writeInstalledMapFiles(t, cfg.Cfg.GetMapsFolderPath(), paths.TilesPath(), "MIS", types.ConfigData{
		Code:    "MIS",
		Name:    "Missing",
		Version: "1.0.0",
		Country: func() *string { value := "US"; return &value }(),
	})

	profile := types.DefaultProfile()
	profile.Subscriptions.Maps["missing-map"] = "1.0.0"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))
	require.Len(t, reg.GetInstalledMaps(), 1)
	require.Equal(t, "missing-map", reg.GetInstalledMaps()[0].ID)
	require.Equal(t, "MIS", reg.GetInstalledMaps()[0].MapConfig.Code)
}

func TestBootstrapInstalledStateFromProfileHydratesLocalMapConfigFromDisk(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	country := "JP"
	cityCode := "KCZ"
	pitch := 35.0
	configData := types.ConfigData{
		Code:        cityCode,
		Name:        "Kochi",
		Description: "Local imported map",
		Population:  340000,
		Creator:     "suscat",
		Country:     &country,
		Version:     "0.9.0",
		InitialViewState: struct {
			Latitude  float64  `json:"latitude"`
			Longitude float64  `json:"longitude"`
			Zoom      float64  `json:"zoom"`
			Pitch     *float64 `json:"pitch,omitempty"`
			Bearing   float64  `json:"bearing"`
		}{
			Latitude:  33.5597,
			Longitude: 133.5311,
			Zoom:      11.5,
			Pitch:     &pitch,
			Bearing:   12,
		},
	}
	mapPath := paths.JoinLocalPath(cfg.Cfg.GetMapsFolderPath(), cityCode)
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	writeInstalledMapFiles(t, cfg.Cfg.GetMapsFolderPath(), paths.TilesPath(), cityCode, configData)
	localMapID := cityCode
	reg.installedMaps = []types.InstalledMapInfo{
		{
			ID:        localMapID,
			Version:   "1.0.0",
			IsLocal:   true,
			MapConfig: configData,
		},
	}

	profile := types.DefaultProfile()
	profile.Subscriptions.LocalMaps[localMapID] = "1.2.3"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))

	installedMaps := reg.GetInstalledMaps()
	require.Len(t, installedMaps, 1)
	require.Equal(t, localMapID, installedMaps[0].ID)
	require.True(t, installedMaps[0].IsLocal)
	require.Equal(t, "1.2.3", installedMaps[0].Version)
	require.Equal(t, "1.2.3", installedMaps[0].MapConfig.Version)
	require.Equal(t, cityCode, installedMaps[0].MapConfig.Code)
	require.Equal(t, configData.Name, installedMaps[0].MapConfig.Name)
	require.Equal(t, configData.Description, installedMaps[0].MapConfig.Description)
	require.Equal(t, configData.Population, installedMaps[0].MapConfig.Population)
	require.Equal(t, configData.Creator, installedMaps[0].MapConfig.Creator)
	require.Equal(t, configData.Country, installedMaps[0].MapConfig.Country)
	require.Equal(t, configData.InitialViewState, installedMaps[0].MapConfig.InitialViewState)
}

func TestBootstrapInstalledStateFromProfileKeepsRemoteMapWhenDownloadedDataFilesExist(t *testing.T) {
	testutil.NewHarness(t)
	country := "IT"
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Maps: []types.MapManifest{
			{
				ID:          "map-a",
				CityCode:    "AAA",
				Name:        "Map A",
				Description: "Map Description",
				Author:      "Author A",
				Country:     country,
				Population:  123456,
			},
		},
	})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	mapPath := paths.JoinLocalPath(cfg.Cfg.MetroMakerDataPath, "cities", "data", "AAA")
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "buildings_index.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "demand_data.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "roads.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "runways_taxiways.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.MkdirAll(paths.TilesPath(), 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(paths.TilesPath(), "AAA.pmtiles"), []byte("tiles"), 0o644))

	profile := types.DefaultProfile()
	profile.Subscriptions.Maps["map-a"] = "2.0.0"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))
	require.Len(t, reg.GetInstalledMaps(), 1)
	require.Equal(t, "map-a", reg.GetInstalledMaps()[0].ID)
	require.Equal(t, "2.0.0", reg.GetInstalledMaps()[0].Version)
}

func TestBootstrapInstalledStateFromProfilePreservesExistingRemoteMapConfigAndBackfillsCountry(t *testing.T) {
	testutil.NewHarness(t)
	country := "IT"
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Maps: []types.MapManifest{
			{
				ID:          "map-a",
				CityCode:    "AAA",
				Name:        "Registry Name",
				Description: "Registry Description",
				Author:      "Registry Author",
				Country:     country,
				Population:  123456,
			},
		},
	})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	mapPath := paths.JoinLocalPath(cfg.Cfg.GetMapsFolderPath(), "AAA")
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "buildings_index.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "demand_data.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "roads.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "runways_taxiways.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.MkdirAll(paths.TilesPath(), 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(paths.TilesPath(), "AAA.pmtiles"), []byte("tiles"), 0o644))

	reg.installedMaps = []types.InstalledMapInfo{
		{
			ID:      "map-a",
			Version: "1.0.0",
			IsLocal: false,
			MapConfig: types.ConfigData{
				Code:        "AAA",
				Name:        "Existing Name",
				Description: "Existing Description",
				Population:  654321,
				Creator:     "Existing Author",
				Version:     "1.0.0",
			},
		},
	}

	profile := types.DefaultProfile()
	profile.Subscriptions.Maps["map-a"] = "2.0.0"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))
	require.Len(t, reg.GetInstalledMaps(), 1)
	mapInfo := reg.GetInstalledMaps()[0]
	require.Equal(t, "map-a", mapInfo.ID)
	require.False(t, mapInfo.IsLocal)
	require.Equal(t, "2.0.0", mapInfo.Version)
	require.Equal(t, "AAA", mapInfo.MapConfig.Code)
	require.Equal(t, "2.0.0", mapInfo.MapConfig.Version)
	require.Equal(t, "Existing Name", mapInfo.MapConfig.Name)
	require.Equal(t, "Existing Description", mapInfo.MapConfig.Description)
	require.Equal(t, 654321, mapInfo.MapConfig.Population)
	require.Equal(t, "Existing Author", mapInfo.MapConfig.Creator)
	require.NotNil(t, mapInfo.MapConfig.Country)
	require.Equal(t, country, *mapInfo.MapConfig.Country)
}

func TestBootstrapInstalledStateFromProfileKeepsLocalMap(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	cityCode := "KCZ"
	mapPath := paths.JoinLocalPath(cfg.Cfg.GetMapsFolderPath(), cityCode)
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "buildings_index.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "demand_data.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "roads.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "runways_taxiways.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.MkdirAll(paths.TilesPath(), 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(paths.TilesPath(), cityCode+".pmtiles"), []byte("tiles"), 0o644))
	country := "JP"
	require.NoError(t, files.WriteJSON(paths.JoinLocalPath(mapPath, "config.json"), "installed map config", types.ConfigData{
		Code:    cityCode,
		Name:    "Kochi",
		Version: "0.0.1",
		Country: &country,
	}))

	localMapID := cityCode
	profile := types.DefaultProfile()
	profile.Subscriptions.LocalMaps[localMapID] = "1.2.3"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))
	installedMaps := reg.GetInstalledMaps()
	require.Len(t, installedMaps, 1)
	require.Equal(t, localMapID, installedMaps[0].ID)
	require.True(t, installedMaps[0].IsLocal)
	require.Equal(t, cityCode, installedMaps[0].MapConfig.Code)
	require.Equal(t, "1.2.3", installedMaps[0].MapConfig.Version)
}

func TestBootstrapInstalledStateFromProfilePrefersExistingInstalledConfigForLocalMap(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{})

	cfg := config.NewConfig(testutil.TestLogSink{})
	testutil.SetValidConfigPaths(t, &cfg.Cfg)
	reg := NewRegistry(testutil.TestLogSink{}, cfg)
	require.NoError(t, reg.fetchFromDisk())

	cityCode := "KCZ"
	mapPath := paths.JoinLocalPath(cfg.Cfg.GetMapsFolderPath(), cityCode)
	require.NoError(t, os.MkdirAll(mapPath, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "buildings_index.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "demand_data.json.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "roads.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, "runways_taxiways.geojson.gz"), []byte("{}"), 0o644))
	require.NoError(t, os.MkdirAll(paths.TilesPath(), 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(paths.TilesPath(), cityCode+".pmtiles"), []byte("tiles"), 0o644))

	diskCountry := "JP"
	require.NoError(t, files.WriteJSON(paths.JoinLocalPath(mapPath, "config.json"), "installed map config", types.ConfigData{
		Code:        cityCode,
		Name:        "Disk Name",
		Description: "Disk Description",
		Population:  100,
		Creator:     "Disk Author",
		Version:     "0.0.1",
		Country:     &diskCountry,
	}))

	existingCountry := "US"
	reg.installedMaps = []types.InstalledMapInfo{
		{
			ID:      cityCode,
			Version: "1.0.0",
			IsLocal: true,
			MapConfig: types.ConfigData{
				Code:        cityCode,
				Name:        "Existing Name",
				Description: "Existing Description",
				Population:  999,
				Creator:     "Existing Author",
				Version:     "1.0.0",
				Country:     &existingCountry,
			},
		},
	}

	profile := types.DefaultProfile()
	profile.Subscriptions.LocalMaps[cityCode] = "1.2.3"

	require.NoError(t, reg.BootstrapInstalledStateFromProfile(profile))

	installedMaps := reg.GetInstalledMaps()
	require.Len(t, installedMaps, 1)
	require.Equal(t, cityCode, installedMaps[0].ID)
	require.True(t, installedMaps[0].IsLocal)
	require.Equal(t, "Existing Name", installedMaps[0].MapConfig.Name)
	require.Equal(t, "Existing Description", installedMaps[0].MapConfig.Description)
	require.Equal(t, 999, installedMaps[0].MapConfig.Population)
	require.Equal(t, "Existing Author", installedMaps[0].MapConfig.Creator)
	require.NotNil(t, installedMaps[0].MapConfig.Country)
	require.Equal(t, existingCountry, *installedMaps[0].MapConfig.Country)
	require.Equal(t, cityCode, installedMaps[0].MapConfig.Code)
	require.Equal(t, "1.2.3", installedMaps[0].MapConfig.Version)
}

func TestInstalledStatePersistsMutations(t *testing.T) {
	testutil.NewHarness(t)
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))

	reg.AddInstalledMod("mod-a", "1.0.0", true)
	reg.AddInstalledMod("mod-b", "2.0.0", false)
	reg.AddInstalledMap("map-a", "1.0.0", true, types.ConfigData{Code: "AAA"})
	reg.AddInstalledMap("map-b", "2.0.0", false, types.ConfigData{Code: "BBB"})

	require.Len(t, reg.GetInstalledMods(), 2)
	require.Len(t, reg.GetInstalledMaps(), 2)
	require.ElementsMatch(t, []string{"AAA", "BBB"}, reg.GetInstalledMapCodes())

	modsResp := reg.GetInstalledModsResponse()
	require.Equal(t, types.ResponseSuccess, modsResp.Status)
	require.Len(t, modsResp.Mods, 2)

	mapsResp := reg.GetInstalledMapsResponse()
	require.Equal(t, types.ResponseSuccess, mapsResp.Status)
	require.Len(t, mapsResp.Maps, 2)

	reg.RemoveInstalledMod("mod-a")
	reg.RemoveInstalledMap("map-a")
	require.Equal(t, []types.InstalledModInfo{
		{ID: "mod-b", Version: "2.0.0", IsLocal: false},
	}, reg.GetInstalledMods())
	require.Equal(t, []types.InstalledMapInfo{
		{ID: "map-b", Version: "2.0.0", IsLocal: false, MapConfig: types.ConfigData{Code: "BBB"}},
	}, reg.GetInstalledMaps())
}

func TestGetRemoteInstalledMaps(t *testing.T) {
	testutil.NewHarness(t)
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig(testutil.TestLogSink{}))

	reg.AddInstalledMap("map-remote", "1.0.0", false, types.ConfigData{Code: "AAA"})
	reg.AddInstalledMap("map-local", "1.2.0", true, types.ConfigData{Code: "BBB"})

	remote := reg.GetRemoteInstalledMaps()
	require.Equal(t, []types.InstalledMapInfo{
		{
			ID:      "map-remote",
			Version: "1.0.0",
			IsLocal: false,
			MapConfig: types.ConfigData{
				Code: "AAA",
			},
		},
	}, remote)
}
