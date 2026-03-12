package registrytest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

type LastUpdatedFixture struct {
	AssetID   string
	AssetType types.AssetType
	Path      string
	Versions  []types.CustomUpdateVersion
	Status    int
}

func MockLastUpdatedServer(t *testing.T, reg any, fixtures []LastUpdatedFixture) func() {
	t.Helper()

	mux := http.NewServeMux()
	mods := make([]types.ModManifest, 0)
	maps := make([]types.MapManifest, 0)

	for _, fixture := range fixtures {
		current := fixture
		mux.HandleFunc(current.Path, func(w http.ResponseWriter, r *http.Request) {
			if current.Status != 0 && current.Status != http.StatusOK {
				http.Error(w, "failed", current.Status)
				return
			}

			payload := types.CustomUpdateFile{
				SchemaVersion: 1,
				Versions:      current.Versions,
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(payload))
		})
	}

	server := httptest.NewServer(mux)

	for _, fixture := range fixtures {
		update := types.UpdateConfig{
			Type: "custom",
			URL:  server.URL + fixture.Path,
		}
		switch fixture.AssetType {
		case types.AssetTypeMap:
			maps = append(maps, types.MapManifest{ID: fixture.AssetID, Update: update})
		case types.AssetTypeMod:
			mods = append(mods, types.ModManifest{ID: fixture.AssetID, Update: update})
		}
	}

	SetUnexportedField(t, reg, "httpClient", server.Client())
	SetManifestsForTest(t, reg, mods, maps)
	return server.Close
}
