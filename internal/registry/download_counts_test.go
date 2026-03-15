package registry

import (
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

func loadedRegistryWithDownloads(t *testing.T) *Registry {
	t.Helper()
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Mods: []types.ModManifest{
			{ID: "mod-a"},
		},
		Maps: []types.MapManifest{
			{ID: "map-a"},
		},
		ModDownloadEntries: map[string]map[string]int{
			"mod-a": {
				"1.0.0": 5,
				"1.1.0": 8,
			},
		},
		MapDownloadEntries: map[string]map[string]int{
			"map-a": {
				"0.1.0": 11,
			},
		},
	})

	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig())
	require.NoError(t, reg.fetchFromDisk())
	return reg
}

func TestFetchFromDiskLoadsDownloadCounts(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	require.Equal(t, 8, reg.downloadCounts[types.AssetTypeMod]["mod-a"]["1.1.0"])
	require.Equal(t, 11, reg.downloadCounts[types.AssetTypeMap]["map-a"]["0.1.0"])
}

func TestGetAssetDownloadCounts(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	result := reg.GetAssetDownloadCounts(types.AssetTypeMap, "map-a")
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "map", result.AssetType)
	require.Equal(t, "map-a", result.AssetID)
	require.Equal(t, 11, result.Counts["0.1.0"])
}

func TestGetAssetDownloadCountsInvalidType(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	result := reg.GetAssetDownloadCounts(types.AssetType("amazing_maps"), "map-a")
	require.Equal(t, types.ResponseError, result.Status)
	require.Contains(t, result.Message, "invalid asset type")
	require.Empty(t, result.Counts)
}

func TestGetAssetDownloadCountsMissingAssetReturnsEmpty(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	result := reg.GetAssetDownloadCounts(types.AssetTypeMod, "missing-mod")
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Empty(t, result.Counts)
}

func TestGetDownloadCountsByAssetType(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	result := reg.GetDownloadCountsByAssetType(types.AssetTypeMod)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, 5, result.Counts["mod-a"]["1.0.0"])
}

func TestGetDownloadCountsByAssetTypeInvalidType(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	result := reg.GetDownloadCountsByAssetType(types.AssetType("terrible_mods"))
	require.Equal(t, types.ResponseError, result.Status)
	require.Contains(t, result.Message, "invalid asset type")
	require.Empty(t, result.Counts)
}

func TestGetDownloadCountsByAssetTypeReturnsDeepCopy(t *testing.T) {
	reg := loadedRegistryWithDownloads(t)

	first := reg.GetDownloadCountsByAssetType(types.AssetTypeMod)
	first.Counts["mod-a"]["1.0.0"] = 999

	second := reg.GetDownloadCountsByAssetType(types.AssetTypeMod)
	require.Equal(t, 5, second.Counts["mod-a"]["1.0.0"])
}

func TestFetchFromDiskFiltersOutAssetsMissingIntegrityListings(t *testing.T) {
	testutil.NewHarness(t)
	registrytest.WriteFixture(t, registrytest.RepositoryFixture{
		Mods: []types.ModManifest{
			{ID: "mod-a"},
			{ID: "mod-b"},
		},
		Maps: []types.MapManifest{
			{ID: "map-a"},
			{ID: "map-b"},
		},
	})

	require.NoError(t, files.WriteJSON(
		filepath.Join(paths.RegistryRepoPath(), "mods", constants.INTEGRITY_JSON),
		"mods integrity report",
		types.RegistryIntegrityReport{
			SchemaVersion: 1,
			GeneratedAt:   "1970-01-01T00:00:00Z",
			Listings: map[string]types.IntegrityListing{
				"mod-a": {Versions: map[string]types.IntegrityVersionStatus{}},
			},
		},
	))
	require.NoError(t, files.WriteJSON(
		filepath.Join(paths.RegistryRepoPath(), "maps", constants.INTEGRITY_JSON),
		"maps integrity report",
		types.RegistryIntegrityReport{
			SchemaVersion: 1,
			GeneratedAt:   "1970-01-01T00:00:00Z",
			Listings: map[string]types.IntegrityListing{
				"map-a": {Versions: map[string]types.IntegrityVersionStatus{}},
			},
		},
	))

	reg := NewRegistry(testutil.TestLogSink{}, config.NewConfig())
	require.NoError(t, reg.fetchFromDisk())

	require.Len(t, reg.GetMods(), 1)
	require.Equal(t, "mod-a", reg.GetMods()[0].ID)
	require.Len(t, reg.GetMaps(), 1)
	require.Equal(t, "map-a", reg.GetMaps()[0].ID)
}
