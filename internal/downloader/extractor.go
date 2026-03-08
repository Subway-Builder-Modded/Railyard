package downloader

import (
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"railyard/internal/files"
	"railyard/internal/types"
	"railyard/internal/utils"
)

// extractMod processes the downloaded mod zip file, extracts it to the appropriate location.
func extractMod(d *Downloader, filePath string, modId string) types.GenericResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.throwError("Failed to open zip file", err, "file_path", filePath, "mod_id", modId)
	}
	defer reader.Close()

	destFolder := path.Join(d.getModPath(), modId)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return d.throwError("Failed to create destination folder", err, "destination", destFolder, "mod_id", modId)
	}

	fileCount := 0
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			fileCount++
		}
	}

	// First pass: create all directories
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			destPath := path.Join(destFolder, file.Name)
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return d.throwError("Failed to create directory", err, "destination", destPath, "mod_id", modId)
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

				destPath := path.Join(destFolder, file.Name)

				// Ensure parent directory exists
				parentDir := path.Dir(destPath)
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
		return d.throwError("Failed to extract file", err, "file_path", filePath, "mod_id", modId)
	}

	return d.successResponse("Mod extracted successfully", "file_path", filePath, "mod_id", modId)
}

// extractMap processes the downloaded map zip file, validates required files, extracts them to the appropriate locations.
func extractMap(d *Downloader, filePath string) types.MapExtractResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.throwMapExtractError("Failed to open zip file", err, "file_path", filePath)
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
		return d.throwMapExtractErrorSimple("Zip file is missing one or more required files", "file_path", filePath)
	}

	filesCount := 0
	for key, fileStruct := range filesFound {
		if fileStruct.Found && key != "config" {
			filesCount++
		}
	}

	var configData types.ConfigData
	configReader, err := filesFound["config"].FileObject.Open()
	if err != nil {
		return d.throwMapExtractError("Failed to read config file", err, "file_path", filePath)
	}
	defer configReader.Close()

	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return d.throwMapExtractError("Failed to read config file", err, "file_path", filePath)
	}

	configData, err = files.ParseJSON[types.ConfigData](configBytes, "config")
	if err != nil {
		return d.throwMapExtractError("Failed to parse config file", err, "file_path", filePath)
	}

	if configData.ThumbnailBbox != nil && !filesFound["thumbnail"].Found {
		filesCount++
	}

	if d.isMapCodeTaken(configData.Code) {
		return d.throwMapExtractErrorSimple("Cannot install map because its code matches a vanilla map included with the game or an already installed map.", "map_code", configData.Code)
	}

	// Create necessary directories first
	destFolder := path.Join(d.getMapDataPath(), configData.Code)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return d.throwMapExtractError("Failed to create destination folder", err, "destination", destFolder)
	}

	if err := os.MkdirAll(d.mapTilePath, os.ModePerm); err != nil {
		return d.throwMapExtractError("Failed to create tiles directory", err, "tiles_path", d.mapTilePath)
	}

	if err := os.MkdirAll(d.getMapThumbnailPath(), os.ModePerm); err != nil {
		return d.throwMapExtractError("Failed to create thumbnail directory", err, "thumbnail_path", d.getMapThumbnailPath())
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
				extractFileMap(path.Join(d.mapTilePath, configData.Code+".pmtiles"), srcFile, errChan, false)
				if d.OnExtractProgress != nil {
					extractCount++
					d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
				}

			case "thumbnail":
				extractFileMap(path.Join(d.getMapThumbnailPath(), configData.Code+".svg"), srcFile, errChan, false)
				if d.OnExtractProgress != nil {
					extractCount++
					d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
				}

			default:
				extractFileMap(path.Join(destFolder, path.Base(fileStruct.FileObject.Name)+".gz"), srcFile, errChan, true)
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
		return d.throwMapExtractError("Failed to extract file", err, "file_path", filePath)
	}

	if !filesFound["thumbnail"].Found {
		srv, port, srvErr := utils.StartTempPMTilesServer()
		if srvErr != nil {
			return d.warnMapExtractResponse("Failed to start PMTiles server for thumbnail generation, but map was extracted successfully.", configData, "file_path", filePath, "map_code", configData.Code)
		}
		defer srv.Close()

		thumbnailData, err := utils.GenerateThumbnail(configData.Code, configData, port)
		if err != nil {
			return d.warnMapExtractResponse("Failed to generate thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", configData, "file_path", filePath, "map_code", configData.Code)
		}

		thumbnailPath := path.Join(d.getMapThumbnailPath(), configData.Code+".svg")
		if err := os.WriteFile(thumbnailPath, []byte(thumbnailData), os.ModePerm); err != nil {
			return d.warnMapExtractResponse("Failed to save generated thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", configData, "file_path", filePath, "map_code", configData.Code, "thumbnail_path", thumbnailPath)
		}
		extractCount++
		if d.OnExtractProgress != nil {
			d.OnExtractProgress(configData.Code, int64(extractCount), int64(filesCount))
		}
	}

	return d.successMapExtractResponse("Map extracted successfully", configData, "file_path", filePath, "map_code", configData.Code)
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
