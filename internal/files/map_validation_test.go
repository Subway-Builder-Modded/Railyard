package files

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/paths"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestValidateMapArchive(t *testing.T) {
	requiredFiles := func(configCode string) map[string][]byte {
		return map[string][]byte{
			MapConfigFileName:    mustMapConfigJSON(t, configCode),
			MapDemandFileName:    []byte("{}"),
			MapRoadsFileName:     []byte("{}"),
			MapRunwaysFileName:   []byte("{}"),
			MapBuildingsFileName: []byte("{}"),
			"AAA.pmtiles":        []byte("tiles"),
		}
	}

	tests := []struct {
		name        string
		files       map[string][]byte
		wantErrType types.DownloaderErrorType
		wantErr     bool
		wantCode    string
	}{
		{
			name:        "valid archive",
			files:       requiredFiles("AAA"),
			wantErrType: "",
			wantErr:     false,
			wantCode:    "AAA",
		},
		{
			name: "missing required file",
			files: func() map[string][]byte {
				f := requiredFiles("AAA")
				delete(f, MapRoadsFileName)
				return f
			}(),
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
		{
			name: "invalid config json",
			files: func() map[string][]byte {
				f := requiredFiles("AAA")
				f[MapConfigFileName] = []byte("{invalid")
				return f
			}(),
			wantErrType: types.InstallErrorInvalidManifest,
			wantErr:     true,
		},
		{
			name:        "invalid map code",
			files:       requiredFiles("dca"),
			wantErrType: types.InstallErrorInvalidMapCode,
			wantErr:     true,
		},
		{
			name: "missing tile file",
			files: func() map[string][]byte {
				f := requiredFiles("AAA")
				delete(f, "AAA.pmtiles")
				return f
			}(),
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipPath := writeZipArchive(t, tt.files)
			config, errType, err := ValidateMapArchive(zipPath)
			require.Equal(t, tt.wantErrType, errType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantCode, config.Code)
		})
	}
}

func TestValidateInstalledMapDataLocal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, mapRoot, tilesRoot, cityCode string)
		wantErrType types.DownloaderErrorType
		wantErr     bool
		wantCode    string
	}{
		{
			name: "valid local installed map data",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "AAA")
			},
			wantErrType: "",
			wantErr:     false,
			wantCode:    "AAA",
		},
		{
			name: "missing config file",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "AAA")
				configPath := paths.JoinLocalPath(mapRoot, cityCode, MapConfigFileName)
				require.NoError(t, os.Remove(configPath))
			},
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
		{
			name: "missing required gz file",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "AAA")
				roadsPath := paths.JoinLocalPath(mapRoot, cityCode, MapRoadsFileName+".gz")
				require.NoError(t, os.Remove(roadsPath))
			},
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
		{
			name: "missing tile file",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "AAA")
				tilePath := paths.JoinLocalPath(tilesRoot, cityCode+MapTileFileExt)
				require.NoError(t, os.Remove(tilePath))
			},
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
		{
			name: "invalid installed config json",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "AAA")
				configPath := paths.JoinLocalPath(mapRoot, cityCode, MapConfigFileName)
				require.NoError(t, os.WriteFile(configPath, []byte("{invalid"), 0o644))
			},
			wantErrType: types.InstallErrorInvalidManifest,
			wantErr:     true,
		},
		{
			name: "invalid installed config map code",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledLocalMapFixture(t, mapRoot, tilesRoot, cityCode, "aaa")
			},
			wantErrType: types.InstallErrorInvalidMapCode,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapRoot := t.TempDir()
			tilesRoot := t.TempDir()
			cityCode := "AAA"
			tt.setup(t, mapRoot, tilesRoot, cityCode)

			config, errType, err := ValidateInstalledMapData(mapRoot, tilesRoot, cityCode, true)
			require.Equal(t, tt.wantErrType, errType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantCode, config.Code)
		})
	}
}

func TestValidateInstalledMapDataDownloaded(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, mapRoot, tilesRoot, cityCode string)
		wantErrType types.DownloaderErrorType
		wantErr     bool
	}{
		{
			name: "valid downloaded installed map data",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledDownloadedMapFixture(t, mapRoot, tilesRoot, cityCode)
			},
			wantErrType: "",
			wantErr:     false,
		},
		{
			name: "missing required file",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledDownloadedMapFixture(t, mapRoot, tilesRoot, cityCode)
				demandPath := paths.JoinLocalPath(mapRoot, cityCode, MapDemandFileName+".gz")
				require.NoError(t, os.Remove(demandPath))
			},
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
		{
			name: "missing tile file",
			setup: func(t *testing.T, mapRoot, tilesRoot, cityCode string) {
				writeInstalledDownloadedMapFixture(t, mapRoot, tilesRoot, cityCode)
				tilePath := paths.JoinLocalPath(tilesRoot, cityCode+MapTileFileExt)
				require.NoError(t, os.Remove(tilePath))
			},
			wantErrType: types.InstallErrorInvalidArchive,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapRoot := t.TempDir()
			tilesRoot := t.TempDir()
			cityCode := "AAA"
			tt.setup(t, mapRoot, tilesRoot, cityCode)

			_, errType, err := ValidateInstalledMapData(mapRoot, tilesRoot, cityCode, false)
			require.Equal(t, tt.wantErrType, errType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func mustMapConfigJSON(t *testing.T, code string) []byte {
	t.Helper()
	cfg := types.ConfigData{
		Code:    code,
		Name:    "Test Map",
		Version: "1.0.0",
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	return data
}

func writeZipArchive(t *testing.T, files map[string][]byte) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "map.zip")
	file, err := os.Create(zipPath)
	require.NoError(t, err)
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	for name, content := range files {
		entry, createErr := zipWriter.Create(name)
		require.NoError(t, createErr)
		_, writeErr := entry.Write(content)
		require.NoError(t, writeErr)
	}
	require.NoError(t, zipWriter.Close())
	return zipPath
}

func writeInstalledDownloadedMapFixture(t *testing.T, mapRoot, tilesRoot, cityCode string) {
	t.Helper()
	cityPath := paths.JoinLocalPath(mapRoot, cityCode)
	require.NoError(t, os.MkdirAll(cityPath, 0o755))
	require.NoError(t, os.MkdirAll(tilesRoot, 0o755))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(cityPath, MapDemandFileName+".gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(cityPath, MapRoadsFileName+".gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(cityPath, MapRunwaysFileName+".gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(cityPath, MapBuildingsFileName+".gz"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(paths.JoinLocalPath(tilesRoot, cityCode+MapTileFileExt), []byte("tiles"), 0o644))
}

func writeInstalledLocalMapFixture(t *testing.T, mapRoot, tilesRoot, cityCode, configCode string) {
	t.Helper()
	writeInstalledDownloadedMapFixture(t, mapRoot, tilesRoot, cityCode)
	configPath := paths.JoinLocalPath(mapRoot, cityCode, MapConfigFileName)
	require.NoError(t, os.WriteFile(configPath, mustMapConfigJSON(t, configCode), 0o644))
}
