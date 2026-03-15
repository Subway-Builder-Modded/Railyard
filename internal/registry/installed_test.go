package registry

import (
	"os"
	"testing"

	"railyard/internal/config"
	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestWriteInstalledToDiskPersistsMapsAndMods(t *testing.T) {
	testutil.NewHarness(t)
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig())
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
	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig())

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
