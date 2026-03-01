package main

import (
	"archive/zip"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"railyard/internal/files"
	"railyard/internal/types"
	"slices"
	"sync"

	"go.yaml.in/yaml/v4"
)

type Downloader struct {
	tempPath    string
	mapTilePath string
	registry    *Registry
	config      *Config
}

func NewDownloader(config *Config, registry *Registry) *Downloader {
	return &Downloader{
		mapTilePath: path.Join(AppDataRoot(), "tiles"),
		tempPath:    path.Join(AppDataRoot(), "temp"),
		registry:    registry,
		config:      config,
	}
}

func (d *Downloader) getModPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "mods")
}

func (d *Downloader) getMapDataPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "cities", "data")
}

func (d *Downloader) getMapTilePath() string {
	return path.Join(AppDataRoot(), "tiles")
}

func (d *Downloader) getMapThumbnailPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "public", "data", "city-maps")
}

func (d *Downloader) InstallMod(modId string, version string) types.GenericResponse {
	if !d.config.GetConfig().Validation.IsValid() {
		return types.GenericResponse{
			Status: "error",
			Message: "Cannot install mod because app config paths are not properly configured. " +
				"Please set valid paths in the config before installing mods.",
		}
	}
	modInfo, err := d.registry.GetMod(modId)
	if err != nil {
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to get mod info from registry: " + err.Error(),
		}
	}

	versions, err := d.registry.GetVersions(modInfo.Update.Type, modInfo.Update.URL)
	if err != nil {
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to get mod versions from registry: " + err.Error(),
		}
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return types.GenericResponse{
			Status:  "error",
			Message: "Specified version not found for mod",
		}
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to download mod zip: " + downloadResp.Message,
		}
	}

	extractResp := d.handleModExtract(downloadResp.Path, modId)
	if extractResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to extract mod zip: " + extractResp.Message,
		}
	}
	os.Remove(downloadResp.Path)
	d.registry.AddInstalledMod(modId, version)
	return types.GenericResponse{
		Status:  "success",
		Message: "Mod installed successfully",
	}
}

func (d *Downloader) InstallMap(mapId string, version string) types.MapExtractResponse {
	if !d.config.GetConfig().Validation.IsValid() {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Invalid configuration",
			},
		}
	}
	mapInfo, err := d.registry.GetMap(mapId)
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to get map info from registry: " + err.Error(),
			},
		}
	}

	versions, err := d.registry.GetVersions(mapInfo.Update.Type, mapInfo.Update.URL)
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to get map versions from registry: " + err.Error(),
			},
		}
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Specified version not found for map",
			},
		}
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to download map zip: " + downloadResp.Message,
			},
		}
	}

	extractResp := d.handleMapExtract(downloadResp.Path)
	if extractResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to extract map zip: " + extractResp.Message,
			},
		}
	}
	os.Remove(downloadResp.Path)
	d.registry.AddInstalledMap(mapId, version, extractResp.Config)
	return extractResp
}

func (d *Downloader) downloadTempZip(url string) types.DownloadTempResponse {
	if err := os.MkdirAll(d.tempPath, os.ModePerm); err != nil {
		return types.DownloadTempResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to create temp directory: " + err.Error(),
			},
		}
	}

	file, err := os.CreateTemp(d.tempPath, "download-*.zip")
	if err != nil {
		return types.DownloadTempResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to create temp file: " + err.Error(),
			},
		}
	}
	defer file.Close()
	zip, err := http.Get(url)

	if err != nil || zip.StatusCode != http.StatusOK {
		return types.DownloadTempResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to download file: " + err.Error(),
			},
		}
	}

	_, err = io.Copy(file, zip.Body)
	if err != nil {
		return types.DownloadTempResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to save file: " + err.Error(),
			},
		}
	}

	return types.DownloadTempResponse{
		GenericResponse: types.GenericResponse{
			Status:  "success",
			Message: "File downloaded successfully",
		},
		Path: file.Name(),
	}
}

func (d *Downloader) handleMapExtract(filePath string) types.MapExtractResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to open zip file: " + err.Error(),
			},
		}
	}

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

	if !filesFound["config"].Found ||
		!filesFound["demandData"].Found ||
		!filesFound["roads"].Found ||
		!filesFound["runways"].Found ||
		!filesFound["buildings"].Found ||
		!filesFound["tiles"].Found {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Zip file is missing one or more required files",
			},
		}
	}

	var configData types.ConfigData
	configReader, err := filesFound["config"].FileObject.Open()
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to read config file: " + err.Error(),
			},
		}
	}

	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to read config file: " + err.Error(),
			},
		}
	}

	configData, err = files.ParseJSON[types.ConfigData](configBytes, "config")
	if err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to parse config file: " + err.Error(),
			},
		}
	}

	if slices.Contains(d.getVanillaMapCodes(), configData.Code) || slices.Contains(d.registry.GetInstalledMapCodes(), configData.Code) {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Cannot install map because its code matches a vanilla map included with the game or an already installed map.",
			},
		}
	}

	// Create necessary directories first
	destFolder := path.Join(d.getMapDataPath(), configData.Code)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to create destination folder: " + err.Error(),
			},
		}
	}

	if err := os.MkdirAll(d.mapTilePath, os.ModePerm); err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to create tiles directory: " + err.Error(),
			},
		}
	}

	if err := os.MkdirAll(d.getMapThumbnailPath(), os.ModePerm); err != nil {
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to create thumbnail directory: " + err.Error(),
			},
		}
	}

	// Process files in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(filesFound))

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
				destPath := path.Join(d.mapTilePath, configData.Code+".pmtiles")
				destFile, err := os.Create(destPath)
				if err != nil {
					errChan <- err
					return
				}
				defer destFile.Close()

				_, err = io.Copy(destFile, srcFile)
				if err != nil {
					errChan <- err
					return
				}

			case "thumbnail":
				destPath := path.Join(d.getMapThumbnailPath(), configData.Code+".svg")
				destFile, err := os.Create(destPath)
				if err != nil {
					errChan <- err
					return
				}
				defer destFile.Close()

				_, err = io.Copy(destFile, srcFile)
				if err != nil {
					errChan <- err
					return
				}

			default:
				// Handle gzipped files (demandData, roads, runways, buildings, oceanDepth)
				destPath := path.Join(destFolder, path.Base(fileStruct.FileObject.Name)+".gz")
				destFile, err := os.Create(destPath)
				if err != nil {
					errChan <- err
					return
				}
				defer destFile.Close()

				gzipWriter := gzip.NewWriter(destFile)
				defer gzipWriter.Close()

				_, err = io.Copy(gzipWriter, srcFile)
				if err != nil {
					errChan <- err
					return
				}
			}
		}(key, fileStruct)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	if len(errChan) > 0 {
		err := <-errChan
		return types.MapExtractResponse{
			GenericResponse: types.GenericResponse{
				Status:  "error",
				Message: "Failed to extract file: " + err.Error(),
			},
		}
	}

	return types.MapExtractResponse{
		GenericResponse: types.GenericResponse{
			Status:  "success",
			Message: "Map extracted successfully",
		},
		Config: configData,
	}
}

func (d *Downloader) handleModExtract(filePath string, modId string) types.GenericResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to open zip file: " + err.Error(),
		}
	}
	defer reader.Close()

	destFolder := path.Join(d.getModPath(), modId)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to create destination folder: " + err.Error(),
		}
	}

	// First pass: create all directories
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			destPath := path.Join(destFolder, file.Name)
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return types.GenericResponse{
					Status:  "error",
					Message: "Failed to create directory: " + err.Error(),
				}
			}
		}
	}

	// Second pass: extract files in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(reader.File))

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
				if err != nil {
					errChan <- err
					return
				}
			}(file)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	if len(errChan) > 0 {
		err := <-errChan
		return types.GenericResponse{
			Status:  "error",
			Message: "Failed to extract file: " + err.Error(),
		}
	}

	return types.GenericResponse{
		Status:  "success",
		Message: "Mod extracted successfully",
	}
}

// getVanillaMapCodes returns the city codes of maps included with the game.
func (d *Downloader) getVanillaMapCodes() []string {
	config := d.config.GetConfig()
	if !config.Validation.IsValid() {
		log.Printf("Warning: Invalid Config: %v", config.Validation)
		return []string{}
	}
	reader, err := os.Open(path.Join(config.Config.MetroMakerDataPath, "cities", "latest-cities.yml"))
	if err != nil {
		log.Printf("Warning: failed to open latest-cities.yml: %v", err)
		return []string{}
	}
	defer reader.Close()

	var citiesData types.CitiesData
	decoder := yaml.NewDecoder(reader)
	err = decoder.Decode(&citiesData)
	if err != nil {
		log.Printf("Warning: failed to parse latest-cities.yml: %v", err)
		return []string{}
	}
	cityCodes := make([]string, 0, len(citiesData.Cities))
	for code := range citiesData.Cities {
		cityCodes = append(cityCodes, code)
	}
	return cityCodes
}
