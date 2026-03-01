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

// NewDownloader creates a new Downloader instance with necessary paths and references.
func NewDownloader(config *Config, registry *Registry) *Downloader {
	return &Downloader{
		mapTilePath: path.Join(AppDataRoot(), "tiles"),
		tempPath:    path.Join(AppDataRoot(), "temp"),
		registry:    registry,
		config:      config,
	}
}

// getModPath returns the filesystem path for installed mods.
func (d *Downloader) getModPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "mods")
}

func throwError(message string, err error) types.GenericResponse {
	return types.GenericResponse{
		Status:  types.ResponseError,
		Message: message + ": " + err.Error(),
	}
}

func throwErrorSimple(message string) types.GenericResponse {
	return types.GenericResponse{
		Status:  types.ResponseError,
		Message: message,
	}
}

func throwDownloadError(message string, err error) types.DownloadTempResponse {
	return types.DownloadTempResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseError,
			Message: message + ": " + err.Error(),
		},
	}
}

func throwDownloadErrorSimple(message string) types.DownloadTempResponse {
	return types.DownloadTempResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseError,
			Message: message,
		},
	}
}

func throwMapExtractError(message string, err error) types.MapExtractResponse {
	return types.MapExtractResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseError,
			Message: message + ": " + err.Error(),
		},
	}
}

func throwMapExtractErrorSimple(message string) types.MapExtractResponse {
	return types.MapExtractResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseError,
			Message: message,
		},
	}
}

// getMapDataPath returns the filesystem path for installed map data.
func (d *Downloader) getMapDataPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "cities", "data")
}

// getMapTilePath returns the filesystem path for installed map tiles.
func (d *Downloader) getMapTilePath() string {
	return path.Join(AppDataRoot(), "tiles")
}

// getMapThumbnailPath returns the filesystem path for installed map thumbnails.
func (d *Downloader) getMapThumbnailPath() string {
	return path.Join(d.config.cfg.MetroMakerDataPath, "public", "data", "city-maps")
}

func requiredFilesPresent(filesFound map[string]types.FileFoundStruct) bool {
	for _, fileStruct := range filesFound {
		if fileStruct.Required && !fileStruct.Found {
			return false
		}
	}
	return true
}

// InstallMod handles the installation of a mod given its ID and version, including downloading, extracting, and updating the registry.
func (d *Downloader) InstallMod(modId string, version string) types.GenericResponse {
	if !d.config.GetConfig().Validation.IsValid() {
		return throwErrorSimple("Cannot install mod because app config paths are not properly configured. " +
			"Please set valid paths in the config before installing mods.")
	}
	modInfo, err := d.registry.GetMod(modId)
	if err != nil {
		return throwError("Failed to get mod info from registry", err)
	}

	versions, err := d.registry.GetVersions(modInfo.Update.Type, modInfo.Update.URL)
	if err != nil {
		return throwError("Failed to get mod versions from registry", err)
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return throwErrorSimple("Specified version not found for mod")
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return throwErrorSimple("Failed to download mod zip: " + downloadResp.Message)
	}

	extractResp := d.handleModExtract(downloadResp.Path, modId)
	if extractResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return throwErrorSimple("Failed to extract mod zip: " + extractResp.Message)
	}
	os.Remove(downloadResp.Path)
	d.registry.AddInstalledMod(modId, version)
	return types.GenericResponse{
		Status:  types.ResponseSuccess,
		Message: "Mod installed successfully",
	}
}

// InstallMap handles the installation of a map given its ID and version, including downloading, extracting, validating files, and updating the registry.
func (d *Downloader) InstallMap(mapId string, version string) types.MapExtractResponse {
	if !d.config.GetConfig().Validation.IsValid() {
		return throwMapExtractErrorSimple("Invalid configuration")
	}
	mapInfo, err := d.registry.GetMap(mapId)
	if err != nil {
		return throwMapExtractError("Failed to get map info from registry", err)
	}

	versions, err := d.registry.GetVersions(mapInfo.Update.Type, mapInfo.Update.URL)
	if err != nil {
		return throwMapExtractError("Failed to get map versions from registry", err)
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return throwMapExtractErrorSimple("Specified version not found for map")
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return throwMapExtractErrorSimple("Failed to download map zip: " + downloadResp.Message)
	}

	extractResp := d.handleMapExtract(downloadResp.Path)
	if extractResp.Status != "success" {
		os.Remove(downloadResp.Path)
		return throwMapExtractErrorSimple("Failed to extract map zip: " + extractResp.Message)
	}
	os.Remove(downloadResp.Path)
	d.registry.AddInstalledMap(mapId, version, extractResp.Config)
	return extractResp
}

// downloadTempZip downloads a zip file from the given URL and saves it to a temporary location, returning the path or an error message.
func (d *Downloader) downloadTempZip(url string) types.DownloadTempResponse {
	if err := os.MkdirAll(d.tempPath, os.ModePerm); err != nil {
		return throwDownloadError("Failed to create temp directory", err)
	}

	file, err := os.CreateTemp(d.tempPath, "download-*.zip")
	if err != nil {
		return throwDownloadError("Failed to create temp file", err)
	}
	defer file.Close()
	zip, err := http.Get(url)

	if err != nil || zip.StatusCode != http.StatusOK {
		return throwDownloadError("Failed to download file", err)
	}

	_, err = io.Copy(file, zip.Body)
	if err != nil {
		return throwDownloadError("Failed to save file", err)
	}

	return types.DownloadTempResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseSuccess,
			Message: "File downloaded successfully",
		},
		Path: file.Name(),
	}
}

// handleMapExtract processes the downloaded map zip file, validates required files, extracts them to the appropriate locations, and returns the map config or an error message.
func (d *Downloader) handleMapExtract(filePath string) types.MapExtractResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return throwMapExtractError("Failed to open zip file", err)
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

	if !requiredFilesPresent(filesFound) {
		return throwMapExtractErrorSimple("Zip file is missing one or more required files")
	}

	var configData types.ConfigData
	configReader, err := filesFound["config"].FileObject.Open()
	if err != nil {
		return throwMapExtractError("Failed to read config file", err)
	}

	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return throwMapExtractError("Failed to read config file", err)
	}

	configData, err = files.ParseJSON[types.ConfigData](configBytes, "config")
	if err != nil {
		return throwMapExtractError("Failed to parse config file", err)
	}

	if slices.Contains(d.getVanillaMapCodes(), configData.Code) || slices.Contains(d.registry.GetInstalledMapCodes(), configData.Code) {
		return throwMapExtractErrorSimple("Cannot install map because its code matches a vanilla map included with the game or an already installed map.")
	}

	// Create necessary directories first
	destFolder := path.Join(d.getMapDataPath(), configData.Code)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return throwMapExtractError("Failed to create destination folder", err)
	}

	if err := os.MkdirAll(d.mapTilePath, os.ModePerm); err != nil {
		return throwMapExtractError("Failed to create tiles directory", err)
	}

	if err := os.MkdirAll(d.getMapThumbnailPath(), os.ModePerm); err != nil {
		return throwMapExtractError("Failed to create thumbnail directory", err)
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
				extractFileMap(path.Join(d.mapTilePath, configData.Code+".pmtiles"), srcFile, errChan, false)

			case "thumbnail":
				extractFileMap(path.Join(d.getMapThumbnailPath(), configData.Code+".svg"), srcFile, errChan, false)

			default:
				extractFileMap(path.Join(destFolder, path.Base(fileStruct.FileObject.Name)+".gz"), srcFile, errChan, true)
			}
		}(key, fileStruct)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	if len(errChan) > 0 {
		err := <-errChan
		return throwMapExtractError("Failed to extract file", err)
	}

	return types.MapExtractResponse{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseSuccess,
			Message: "Map extracted successfully",
		},
		Config: configData,
	}
}

// handleModExtract processes the downloaded mod zip file, extracts it to the appropriate location, and returns a success or error message.
func (d *Downloader) handleModExtract(filePath string, modId string) types.GenericResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return throwError("Failed to open zip file", err)
	}
	defer reader.Close()

	destFolder := path.Join(d.getModPath(), modId)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return throwError("Failed to create destination folder", err)
	}

	// First pass: create all directories
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			destPath := path.Join(destFolder, file.Name)
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return throwError("Failed to create directory", err)
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
		return throwError("Failed to extract file", err)
	}

	return types.GenericResponse{
		Status:  types.ResponseSuccess,
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
