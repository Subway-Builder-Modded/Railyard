package registrytest

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

type UpdateFixture struct {
	AssetID      string
	AssetType    types.AssetType
	Versions     []string
	MapCode      string
	FailVersions bool
	// MissingModManifest serves a mod zip without manifest.json to exercise invalid-archive paths.
	MissingModManifest bool
}

func MockZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	tempZip := filepath.Join(t.TempDir(), "fixture.zip")
	f, err := os.Create(tempZip)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	for name, content := range files {
		entry, createErr := w.Create(name)
		require.NoError(t, createErr)
		_, writeErr := entry.Write(content)
		require.NoError(t, writeErr)
	}
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	data, err := os.ReadFile(tempZip)
	require.NoError(t, err)
	return data
}

func MockModZip(t *testing.T) []byte {
	t.Helper()
	manifest, err := json.Marshal(types.MetroMakerModManifest{
		Id:          "fixture-mod",
		Name:        "Fixture Mod",
		Description: "Fixture mod for tests",
		Version:     "1.0.0",
		Main:        "index.js",
	})
	require.NoError(t, err)

	return MockZip(t, map[string][]byte{
		"manifest.json": manifest,
		"index.js":      []byte("export default {};"),
	})
}

func MockModZipMissingManifest(t *testing.T) []byte {
	t.Helper()
	return MockZip(t, map[string][]byte{
		"index.js": []byte("export default {};"),
	})
}

func MockMapZip(t *testing.T, code string) []byte {
	t.Helper()
	configJSON, err := json.Marshal(types.ConfigData{
		Code: code,
		Name: "Fixture Map",
	})
	require.NoError(t, err)

	return MockZip(t, map[string][]byte{
		"config.json":              configJSON,
		"demand_data.json":         []byte("{}"),
		"roads.geojson":            []byte(`{"type":"FeatureCollection","features":[]}`),
		"runways_taxiways.geojson": []byte(`{"type":"FeatureCollection","features":[]}`),
		"buildings_index.json":     []byte("{}"),
		"tiles.pmtiles":            []byte("tiles"),
		"thumbnail.svg":            []byte("<svg></svg>"),
	})
}

func MockRegistryServer(t *testing.T, reg any, fixtures []UpdateFixture) func() {
	t.Helper()
	zipByDownloadPath := map[string][]byte{}
	mods := []types.ModManifest{}
	maps := []types.MapManifest{}
	handler := http.NewServeMux()

	for _, fixture := range fixtures {
		current := fixture
		updatePath := "/updates/" + current.AssetID + ".json"

		if current.AssetType == types.AssetTypeMap {
			maps = append(maps, types.MapManifest{
				ID:     current.AssetID,
				Update: types.UpdateConfig{Type: "custom", URL: "{{BASE_URL}}" + updatePath},
			})
		} else {
			mods = append(mods, types.ModManifest{
				ID:     current.AssetID,
				Update: types.UpdateConfig{Type: "custom", URL: "{{BASE_URL}}" + updatePath},
			})
		}

		handler.HandleFunc(updatePath, func(w http.ResponseWriter, r *http.Request) {
			if current.FailVersions {
				http.Error(w, "failed to fetch versions", http.StatusInternalServerError)
				return
			}

			payload := types.CustomUpdateFile{
				SchemaVersion: 1,
				Versions:      make([]types.CustomUpdateVersion, 0, len(current.Versions)),
			}

			for _, version := range current.Versions {
				downloadPath := "/downloads/" + current.AssetID + "-" + version + ".zip"
				payload.Versions = append(payload.Versions, types.CustomUpdateVersion{
					Version:  version,
					Download: "http://" + r.Host + downloadPath,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(payload))
		})
	}

	for _, fixture := range fixtures {
		current := fixture
		if current.FailVersions {
			continue
		}
		for _, version := range current.Versions {
			downloadPath := "/downloads/" + current.AssetID + "-" + version + ".zip"
			if current.AssetType == types.AssetTypeMap {
				mapCode := current.MapCode
				if mapCode == "" {
					mapCode = "AAA"
				}
				zipByDownloadPath[downloadPath] = MockMapZip(t, mapCode)
				continue
			}
			if current.MissingModManifest {
				zipByDownloadPath[downloadPath] = MockModZipMissingManifest(t)
				continue
			}
			zipByDownloadPath[downloadPath] = MockModZip(t)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if content, ok := zipByDownloadPath[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(content)
			return
		}
		handler.ServeHTTP(w, r)
	}))

	for i := range mods {
		mods[i].Update.URL = strings.ReplaceAll(mods[i].Update.URL, "{{BASE_URL}}", server.URL)
	}
	for i := range maps {
		maps[i].Update.URL = strings.ReplaceAll(maps[i].Update.URL, "{{BASE_URL}}", server.URL)
	}

	SetManifestsForTest(t, reg, mods, maps)
	return server.Close
}
