package registry

import (
	"net/http"
	"testing"
	"time"

	"railyard/internal/testutil/registrytest"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func mustUnix(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed.Unix()
}

func TestResolveLastUpdated(t *testing.T) {
	reg := NewRegistry(testLogSink{})
	closeServer := registrytest.MockLastUpdatedServer(t, reg, []registrytest.LastUpdatedFixture{
		{
			AssetID:   "mod-a",
			AssetType: types.AssetTypeMod,
			Path:      "/updates/mod-a.json",
			Versions: []types.CustomUpdateVersion{
				{Version: "1.0.0", Date: "2025-01-01"},
				{Version: "1.1.0", Date: "2025-02-01"},
			},
			Status: http.StatusOK,
		},
		{
			AssetID:   "map-a",
			AssetType: types.AssetTypeMap,
			Path:      "/updates/map-a.json",
			Versions: []types.CustomUpdateVersion{
				{Version: "2.0.0", Date: "2025-03-01"},
			},
			Status: http.StatusOK,
		},
	})
	defer closeServer()

	modLastUpdated, mapLastUpdated := reg.loadLastUpdated(reg.mods, reg.maps)
	updateManifestLastUpdated(reg.mods, reg.maps, modLastUpdated, mapLastUpdated)

	require.Equal(t, mustUnix(t, "2025-02-01T00:00:00Z"), reg.mods[0].LastUpdated)
	require.Equal(t, mustUnix(t, "2025-03-01T00:00:00Z"), reg.maps[0].LastUpdated)
}

func TestDetermineLatestTimestampWithStable(t *testing.T) {
	versions := []types.VersionInfo{
		{Version: "2.0.0", Date: "2026-01-02T00:00:00Z", Prerelease: true},
		{Version: "1.5.0", Date: "2026-01-01T00:00:00Z", Prerelease: false},
	}

	latest, err := determineLatestTimestamp(testLogSink{}, versions, "github")
	require.NoError(t, err)
	require.Equal(t, mustUnix(t, "2026-01-01T00:00:00Z"), latest)
}

func TestDetermineLatestTimestampFallbackToPreRelease(t *testing.T) {
	versions := []types.VersionInfo{
		{Version: "1.0.0", Date: "2026-01-01T00:00:00Z", Prerelease: true},
		{Version: "1.0.1", Date: "2026-01-03T00:00:00Z", Prerelease: true},
	}

	latest, err := determineLatestTimestamp(testLogSink{}, versions, "github")
	require.NoError(t, err)
	require.Equal(t, mustUnix(t, "2026-01-03T00:00:00Z"), latest)
}

func TestDetermineLatestTimestampRejectsWrongLayout(t *testing.T) {
	githubVersions := []types.VersionInfo{{Version: "1.0.0", Date: "2026-01-01"}}
	_, githubErr := determineLatestTimestamp(testLogSink{}, githubVersions, "github")
	require.Error(t, githubErr)

	customVersions := []types.VersionInfo{{Version: "1.0.0", Date: "2026-01-01T00:00:00Z"}}
	_, customErr := determineLatestTimestamp(testLogSink{}, customVersions, "custom")
	require.Error(t, customErr)
}

func TestLoadLastUpdatedFallsBackToEpochOnFailures(t *testing.T) {
	reg := NewRegistry(testLogSink{})
	closeServer := registrytest.MockLastUpdatedServer(t, reg, []registrytest.LastUpdatedFixture{
		{
			AssetID:   "mod-bad",
			AssetType: types.AssetTypeMod,
			Path:      "/updates/mod-bad.json",
			Status:    http.StatusInternalServerError,
		},
		{
			AssetID:   "map-bad",
			AssetType: types.AssetTypeMap,
			Path:      "/updates/map-bad.json",
			Status:    http.StatusInternalServerError,
		},
	})
	defer closeServer()

	modLastUpdated, mapLastUpdated := reg.loadLastUpdated(reg.mods, reg.maps)
	require.Equal(t, int64(0), modLastUpdated["mod-bad"])
	require.Equal(t, int64(0), mapLastUpdated["map-bad"])
}
