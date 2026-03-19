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
