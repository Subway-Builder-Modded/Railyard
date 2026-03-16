package downloader

import (
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"

	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/types"
	"railyard/internal/utils"
)

// extractMod processes the downloaded mod zip file, extracts it to the appropriate location.
func extractMod(d *Downloader, filePath string, modId string, version string) types.AssetInstallResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Failed to open zip file", err, "file_path", filePath, "mod_id", modId)
	}
	defer reader.Close()

	destFolder := paths.JoinLocalPath(d.getModPath(), modId)

	requiredFiles := map[string]types.FileFoundStruct{
		"manifest":        {Found: false, FileObject: nil, Required: true},
		"manifest_target": {Found: false, FileObject: nil, Required: true},
	}

	fileCount := 0
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			fileCount++
		}
	}

	for _, file := range reader.File {
		if file.Name == constants.MANIFEST_JSON {
			requiredFiles["manifest"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		}
	}

	if !requiredFiles["manifest"].Found {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Zip file is missing manifest.json", nil, "file_path", filePath, "mod_id", modId)
	}

	rawManifestReader, err := requiredFiles["manifest"].FileObject.Open()
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to read manifest file", err, "file_path", filePath, "mod_id", modId)
	}

	rawManifestBytes, err := io.ReadAll(rawManifestReader)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to read manifest file", err, "file_path", filePath, "mod_id", modId)
	}

	manifestData, err := files.ParseJSON[types.MetroMakerModManifest](rawManifestBytes, constants.MANIFEST_JSON)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to parse manifest file", err, "file_path", filePath, "mod_id", modId)
	}
	for _, file := range reader.File {
		if file.Name == manifestData.Main {
			requiredFiles["manifest_target"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
			break
		}
	}

	if !requiredFilesPresent(requiredFiles) {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Zip file is missing one or more required files", nil, "file_path", filePath, "mod_id", modId)
	}

	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorFilesystem, "Failed to create destination folder", err, "destination", destFolder, "mod_id", modId)
	}

	// First pass: create directories to avoid extract errors
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			destPath := paths.JoinLocalPath(destFolder, file.Name)
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorFilesystem, "Failed to create directory during extraction", err, "directory_path", destPath, "mod_id", modId)
			}
		}
	}

	// Second pass: extract files in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(reader.File))

	if d.OnExtractProgress != nil {
		d.OnExtractProgress(modId, 0, int64(fileCount))
	}
	var installCounter atomic.Int64
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			wg.Add(1)
			go func(file *zip.File) {
				defer wg.Done()

				destPath := paths.JoinLocalPath(destFolder, file.Name)

				// Ensure parent directory exists
				parentDir := filepath.Dir(destPath)
				if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
					errChan <- err
					return
				}

				destFile, err := os.Create(destPath)
				if err != nil {
					errChan <- err
					return
				}
				defer destFile.Close()

				srcFile, err := file.Open()
				if err != nil {
					errChan <- err
					return
				}
				defer srcFile.Close()

				_, err = io.Copy(destFile, srcFile)
				if d.OnExtractProgress != nil {
					d.OnExtractProgress(modId, installCounter.Add(1), int64(fileCount))
				}
				if err != nil {
					errChan <- err
					return
				}
			}(file)
		}
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		err := <-errChan
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorExtractFailed, "Failed to extract file", err, "file_path", filePath, "mod_id", modId)
	}

	if _, err := os.Create(paths.JoinLocalPath(destFolder, ".railyard_asset")); err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorFilesystem, "Failed to create asset marker file", err, "mod_id", modId)
	}

	return d.installSuccess(types.AssetTypeMod, modId, version, types.ConfigData{}, "Mod extracted successfully", "file_path", filePath, "assetId", modId)
}

// extractMap processes the downloaded map zip file, validates required files, extracts them to the appropriate locations.
func extractMap(d *Downloader, filePath string, mapId string, version string) types.AssetInstallResponse {
	configData := types.ConfigData{}
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidArchive, "Failed to open zip file", err, "file_path", filePath)
	}
	defer reader.Close()

	filesFound := map[string]types.FileFoundStruct{
		"config":     {Found: false, FileObject: nil, Required: true},
		"demandData": {Found: false, FileObject: nil, Required: true},
		"roads":      {Found: false, FileObject: nil, Required: true},
		"runways":    {Found: false, FileObject: nil, Required: true},
		"buildings":  {Found: false, FileObject: nil, Required: true},
		"tiles":      {Found: false, FileObject: nil, Required: true},
		"oceanDepth": {Found: false, FileObject: nil, Required: false},
		"thumbnail":  {Found: false, FileObject: nil, Required: false},
	}

	for _, file := range reader.File {
		switch file.Name {
		case "config.json":
			filesFound["config"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case "demand_data.json":
			filesFound["demandData"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case "roads.geojson":
			filesFound["roads"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case "runways_taxiways.geojson":
			filesFound["runways"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case "buildings_index.json":
			filesFound["buildings"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		case "ocean_depth_index.json":
			filesFound["oceanDepth"] = types.FileFoundStruct{Found: true, FileObject: file, Required: false}
		}
		if path.Ext(file.Name) == ".pmtiles" {
			filesFound["tiles"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		}
		if path.Ext(file.Name) == ".svg" {
			filesFound["thumbnail"] = types.FileFoundStruct{Found: true, FileObject: file, Required: false}
		}
	}

	if !requiredFilesPresent(filesFound) {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidArchive, "Zip file is missing one or more required files", nil, "file_path", filePath)
	}

	filesCount := 0
	for key, fileStruct := range filesFound {
		if fileStruct.Found && key != "config" {
			filesCount++
		}
	}

	configReader, err := filesFound["config"].FileObject.Open()
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidManifest, "Failed to read config file", err, "file_path", filePath)
	}
	defer configReader.Close()

	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidManifest, "Failed to read config file", err, "file_path", filePath)
	}

	configData, err = files.ParseJSON[types.ConfigData](configBytes, "config")
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidManifest, "Failed to parse config file", err, "file_path", filePath)
	}

	if configData.ThumbnailBbox != nil && !filesFound["thumbnail"].Found {
		filesCount++
	}

	if d.isMapCodeTaken(configData.Code) {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorMapCodeConflict, "Cannot install map because its code matches a vanilla map included with the game or an already installed map.", nil, "map_code", configData.Code)
	}

	// Create necessary directories first
	destFolder := paths.JoinLocalPath(d.getMapDataPath(), configData.Code)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorFilesystem, "Failed to create destination folder", err, "destination", destFolder)
	}

	if err := os.MkdirAll(d.mapTilePath, os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorFilesystem, "Failed to create tiles directory", err, "tiles_path", d.mapTilePath)
	}

	if err := os.MkdirAll(d.getMapThumbnailPath(), os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorFilesystem, "Failed to create thumbnail directory", err, "thumbnail_path", d.getMapThumbnailPath())
	}

	// Process files in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(filesFound))

	extractCount := 0
	if d.OnExtractProgress != nil {
		d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
	}
	for key, fileStruct := range filesFound {
		if !fileStruct.Found || key == "config" {
			continue
		}

		wg.Add(1)
		go func(key string, fileStruct types.FileFoundStruct) {
			defer wg.Done()

			srcFile, err := fileStruct.FileObject.Open()
			if err != nil {
				errChan <- err
				return
			}
			defer srcFile.Close()

			switch key {
			case "tiles":
				extractFileMap(paths.JoinLocalPath(d.mapTilePath, configData.Code+".pmtiles"), srcFile, errChan, false)
				if d.OnExtractProgress != nil {
					extractCount++
					d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
				}

			case "thumbnail":
				extractFileMap(paths.JoinLocalPath(d.getMapThumbnailPath(), configData.Code+".svg"), srcFile, errChan, false)
				if d.OnExtractProgress != nil {
					extractCount++
					d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
				}

			default:
				extractFileMap(paths.JoinLocalPath(destFolder, path.Base(fileStruct.FileObject.Name)+".gz"), srcFile, errChan, true)
				if d.OnExtractProgress != nil {
					extractCount++
					d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
				}
			}
		}(key, fileStruct)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		err := <-errChan
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorExtractFailed, "Failed to extract file", err, "file_path", filePath)
	}

	if !filesFound["thumbnail"].Found {
		srv, port, srvErr := utils.StartTempPMTilesServer()
		if srvErr != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, "Failed to start PMTiles server for thumbnail generation, but map was extracted successfully.", "file_path", filePath, "map_code", configData.Code)
		}
		defer srv.Close()

		thumbnailData, err := utils.GenerateThumbnail(configData.Code, configData, port)
		if err != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, "Failed to generate thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", "file_path", filePath, "map_code", configData.Code)
		}

		thumbnailPath := paths.JoinLocalPath(d.getMapThumbnailPath(), configData.Code+".svg")
		if err := files.WriteFilesAtomically([]files.AtomicFileWrite{
			{
				Path:  thumbnailPath,
				Label: "map thumbnail",
				Data:  []byte(thumbnailData),
				Perm:  0o644,
			},
		}); err != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, "Failed to save generated thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", "file_path", filePath, "map_code", configData.Code, "thumbnail_path", thumbnailPath)
		}
		extractCount++
		if d.OnExtractProgress != nil {
			d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
		}
	}

	if _, err := os.Create(paths.JoinLocalPath(destFolder, ".railyard_asset")); err != nil {
		return d.installError(types.AssetTypeMod, mapId, version, types.ConfigData{}, types.InstallErrorFilesystem, "Failed to create asset marker file", err, "assetId", mapId)
	}

	return d.installSuccess(types.AssetTypeMap, mapId, version, configData, "Map extracted successfully", "file_path", filePath, "map_code", configData.Code)
}

func extractFileMap(path string, srcFile io.ReadCloser, errChan chan<- error, useGzip bool) {
	destFile, err := os.Create(path)
	if err != nil {
		errChan <- err
		return
	}

	defer destFile.Close()

	if useGzip {
		gzipWriter := gzip.NewWriter(destFile)
		defer gzipWriter.Close()
		_, err = io.Copy(gzipWriter, srcFile)
		if err != nil {
			errChan <- err
			return
		}
	} else {
		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			errChan <- err
			return
		}
	}
}
