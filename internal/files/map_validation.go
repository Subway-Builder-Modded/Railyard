package files

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"railyard/internal/paths"
	"railyard/internal/types"
)

const (
	MapConfigFileName     = "config.json"
	MapDemandFileName     = "demand_data.json"
	MapRoadsFileName      = "roads.geojson"
	MapRunwaysFileName    = "runways_taxiways.geojson"
	MapBuildingsFileName  = "buildings_index.json"
	MapOceanDepthFileName = "ocean_depth_index.json"

	MapTileFileExt      = ".pmtiles"
	MapThumbnailFileExt = ".svg"

	MapArchiveKeyConfig     = "config"
	MapArchiveKeyDemandData = "demandData"
	MapArchiveKeyRoads      = "roads"
	MapArchiveKeyRunways    = "runways"
	MapArchiveKeyBuildings  = "buildings"
	MapArchiveKeyTiles      = "tiles"
	MapArchiveKeyThumbnail  = "thumbnail"
	MapArchiveKeyOceanDepth = "oceanDepth"
)

// BuildMapArchiveFileIndex builds an index of expected map archive files for validation, returning a map of file keys to their presence and file objects in the archive
func BuildMapArchiveFileIndex(zipFiles []*zip.File) map[string]types.FileFoundStruct {
	filesFound := map[string]types.FileFoundStruct{
		MapArchiveKeyConfig:     {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyDemandData: {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyRoads:      {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyRunways:    {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyBuildings:  {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyTiles:      {Found: false, FileObject: nil, Required: true},
		MapArchiveKeyThumbnail:  {Found: false, FileObject: nil, Required: false},
		MapArchiveKeyOceanDepth: {Found: false, FileObject: nil, Required: false},
	}

	for _, file := range zipFiles {
		switch file.Name {
		case MapConfigFileName:
			filesFound[MapArchiveKeyConfig] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case MapDemandFileName:
			filesFound[MapArchiveKeyDemandData] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case MapRoadsFileName:
			filesFound[MapArchiveKeyRoads] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case MapRunwaysFileName:
			filesFound[MapArchiveKeyRunways] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case MapBuildingsFileName:
			filesFound[MapArchiveKeyBuildings] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case MapOceanDepthFileName:
			filesFound[MapArchiveKeyOceanDepth] = types.FileFoundStruct{Found: true, FileObject: file, Required: false}
		}
		if path.Ext(file.Name) == MapTileFileExt {
			filesFound[MapArchiveKeyTiles] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		}
		if path.Ext(file.Name) == MapThumbnailFileExt {
			filesFound[MapArchiveKeyThumbnail] = types.FileFoundStruct{Found: true, FileObject: file, Required: false}
		}
	}

	return filesFound
}

// ValidateMapArchive validates required map archive files and parses config.json.
func ValidateMapArchive(filePath string) (types.ConfigData, types.DownloaderErrorType, error) {
	configData := types.ConfigData{}
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return configData, types.InstallErrorInvalidArchive, err
	}
	defer reader.Close()

	filesFound := BuildMapArchiveFileIndex(reader.File)

	if !requiredFilesPresent(filesFound) {
		return configData, types.InstallErrorInvalidArchive, &types.MissingFilesError{Files: []string{"The map archive is missing one or more required files."}}
	}

	configReader, err := filesFound[MapArchiveKeyConfig].FileObject.Open()
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, err
	}
	defer configReader.Close()

	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, err
	}

	configData, err = ParseJSON[types.ConfigData](configBytes, "config")
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, err
	}
	if !types.LocalMapCodePattern.MatchString(configData.Code) {
		return configData, types.InstallErrorInvalidMapCode, fmt.Errorf("invalid map code %q in config.json: must match ^[A-Z]{2,4}$", configData.Code)
	}

	return configData, "", nil
}

func readInstalledMapConfig(mapInstallRoot string, cityCode string) (types.ConfigData, types.DownloaderErrorType, error) {
	configData := types.ConfigData{}
	plainPath := paths.JoinLocalPath(mapInstallRoot, cityCode, MapConfigFileName)
	file, err := os.Open(plainPath)
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, fmt.Errorf("failed to open installed map config: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, fmt.Errorf("failed to read installed map config payload: %w", err)
	}
	configData, err = ParseJSON[types.ConfigData](data, "installed map config")
	if err != nil {
		return configData, types.InstallErrorInvalidManifest, fmt.Errorf("failed to parse installed map config: %w", err)
	}
	if !types.LocalMapCodePattern.MatchString(configData.Code) {
		return configData, types.InstallErrorInvalidMapCode, fmt.Errorf("invalid map code %q in installed map config: must match ^[A-Z]{2,4}$", configData.Code)
	}

	return configData, "", nil
}

// ValidateInstalledMapData validates installed map files under a city-code folder.
// For local maps (isLocal=true), config.json is required and parsed.
// For downloaded maps (isLocal=false), only the compressed city-data files are required.
func ValidateInstalledMapData(mapInstallRoot string, mapTilesRoot string, cityCode string, isLocal bool) (types.ConfigData, types.DownloaderErrorType, error) {
	if isLocal {
		configPath := paths.JoinLocalPath(mapInstallRoot, cityCode, MapConfigFileName)
		if _, err := os.Stat(configPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return types.ConfigData{}, types.InstallErrorInvalidArchive, &types.MissingFilesError{Files: []string{fmt.Sprintf("missing installed map file: %s", configPath)}}
			}
			return types.ConfigData{}, types.InstallErrorFilesystem, fmt.Errorf("failed to stat installed map file %q: %w", configPath, err)
		}
	}

	if errorType, err := validateRequiredInstalledMapFiles(mapInstallRoot, mapTilesRoot, cityCode); err != nil {
		return types.ConfigData{}, errorType, err
	}

	if !isLocal {
		return types.ConfigData{}, "", nil
	}

	return readInstalledMapConfig(mapInstallRoot, cityCode)
}

func requiredFilesPresent(filesFound map[string]types.FileFoundStruct) bool {
	for _, fileStruct := range filesFound {
		if fileStruct.Required && !fileStruct.Found {
			return false
		}
	}
	return true
}

func validateRequiredInstalledMapFiles(mapInstallRoot string, mapTilesRoot string, cityCode string) (types.DownloaderErrorType, error) {
	requiredPaths := []string{
		paths.JoinLocalPath(mapInstallRoot, cityCode, MapDemandFileName+".gz"),
		paths.JoinLocalPath(mapInstallRoot, cityCode, MapRoadsFileName+".gz"),
		paths.JoinLocalPath(mapInstallRoot, cityCode, MapRunwaysFileName+".gz"),
		paths.JoinLocalPath(mapInstallRoot, cityCode, MapBuildingsFileName+".gz"),
		paths.JoinLocalPath(mapTilesRoot, cityCode+MapTileFileExt),
	}

	for _, filePath := range requiredPaths {
		if _, err := os.Stat(filePath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return types.InstallErrorInvalidArchive, &types.MissingFilesError{Files: []string{fmt.Sprintf("missing installed map file: %s", filePath)}}
			}
			return types.InstallErrorFilesystem, fmt.Errorf("failed to stat installed map file %q: %w", filePath, err)
		}
	}
	return "", nil
}
