package downloader

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"slices"

	"railyard/internal/config"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/types"

	"go.yaml.in/yaml/v4"
)

type Downloader struct {
	tempPath    string
	mapTilePath string
	Registry    *registry.Registry
	Config      *config.Config
	Logger      logger.Logger
}

// NewDownloader creates a new Downloader instance with necessary paths and references.
func NewDownloader(cfg *config.Config, reg *registry.Registry, l logger.Logger) *Downloader {
	return &Downloader{
		mapTilePath: path.Join(paths.AppDataRoot(), "tiles"),
		tempPath:    path.Join(paths.AppDataRoot(), "temp"),
		Registry:    reg,
		Config:      cfg,
		Logger:      l,
	}
}

// getModPath returns the filesystem path for installed mods.
func (d *Downloader) getModPath() string {
	return path.Join(d.Config.Cfg.MetroMakerDataPath, "mods")
}

func (d *Downloader) withError(message string, err error) string {
	if err == nil {
		return message
	}
	return message + ": " + err.Error()
}

func (d *Downloader) newGenericResponse(status types.Status, message string, attrs ...any) types.GenericResponse {
	response := types.GenericResponse{Status: status, Message: message}
	if d.Logger != nil {
		d.Logger.LogResponse("Downloader response", response, attrs...)
	}
	return response
}

func (d *Downloader) newDownloadResponse(status types.Status, message string, path string, attrs ...any) types.DownloadTempResponse {
	response := types.DownloadTempResponse{
		GenericResponse: d.newGenericResponse(status, message, attrs...),
		Path:            path,
	}
	return response
}

func (d *Downloader) newMapExtractResponse(status types.Status, message string, config types.ConfigData, attrs ...any) types.MapExtractResponse {
	response := types.MapExtractResponse{
		GenericResponse: d.newGenericResponse(status, message, attrs...),
		Config:          config,
	}
	return response
}

func (d *Downloader) throwError(message string, err error, attrs ...any) types.GenericResponse {
	return d.newGenericResponse(types.ResponseError, d.withError(message, err), attrs...)
}

func (d *Downloader) throwErrorSimple(message string, attrs ...any) types.GenericResponse {
	return d.newGenericResponse(types.ResponseError, message, attrs...)
}

func (d *Downloader) throwDownloadError(message string, err error, attrs ...any) types.DownloadTempResponse {
	return d.newDownloadResponse(types.ResponseError, d.withError(message, err), "", attrs...)
}

func (d *Downloader) throwDownloadErrorSimple(message string, attrs ...any) types.DownloadTempResponse {
	return d.newDownloadResponse(types.ResponseError, message, "", attrs...)
}

func (d *Downloader) throwMapExtractError(message string, err error, attrs ...any) types.MapExtractResponse {
	return d.newMapExtractResponse(types.ResponseError, d.withError(message, err), types.ConfigData{}, attrs...)
}

func (d *Downloader) throwMapExtractErrorSimple(message string, attrs ...any) types.MapExtractResponse {
	return d.newMapExtractResponse(types.ResponseError, message, types.ConfigData{}, attrs...)
}

func (d *Downloader) successResponse(message string, attrs ...any) types.GenericResponse {
	return d.newGenericResponse(types.ResponseSuccess, message, attrs...)
}

func (d *Downloader) warnResponse(message string, attrs ...any) types.GenericResponse {
	return d.newGenericResponse(types.ResponseWarn, message, attrs...)
}

func (d *Downloader) successDownloadResponse(message string, path string, attrs ...any) types.DownloadTempResponse {
	return d.newDownloadResponse(types.ResponseSuccess, message, path, attrs...)
}

func (d *Downloader) successMapExtractResponse(message string, config types.ConfigData, attrs ...any) types.MapExtractResponse {
	return d.newMapExtractResponse(types.ResponseSuccess, message, config, attrs...)
}

func (d *Downloader) warnMapExtractResponse(message string, config types.ConfigData, attrs ...any) types.MapExtractResponse {
	return d.newMapExtractResponse(types.ResponseWarn, message, config, attrs...)
}

// getMapDataPath returns the filesystem path for installed map data.
func (d *Downloader) getMapDataPath() string {
	return path.Join(d.Config.Cfg.MetroMakerDataPath, "cities", "data")
}

// getMapTilePath returns the filesystem path for installed map tiles.
func (d *Downloader) getMapTilePath() string {
	return path.Join(paths.AppDataRoot(), "tiles")
}

// getMapThumbnailPath returns the filesystem path for installed map thumbnails.
func (d *Downloader) getMapThumbnailPath() string {
	return path.Join(d.Config.Cfg.MetroMakerDataPath, "public", "data", "city-maps")
}

func (d *Downloader) UninstallMod(modId string) types.GenericResponse {
	installedMods := d.Registry.GetInstalledMods()
	foundMod := false
	for _, mod := range installedMods {
		if mod.ID == modId {
			foundMod = true
			break
		}
	}
	if !foundMod {
		return d.warnResponse("Mod with ID "+modId+" is not currently installed. No action taken.", "mod_id", modId)
	}
	modPath := path.Join(d.getModPath(), modId)
	if err := os.RemoveAll(modPath); err != nil {
		return d.throwError("Failed to remove mod files", err, "mod_id", modId)
	}
	d.Registry.RemoveInstalledMod(modId)
	return d.successResponse("Mod uninstalled successfully", "mod_id", modId)
}

func (d *Downloader) UninstallMap(mapId string) types.GenericResponse {
	installedMaps := d.Registry.GetInstalledMaps()
	var mapConfig *types.ConfigData = nil
	for _, m := range installedMaps {
		if m.ID == mapId {
			mapConfig = &m.MapConfig
			break
		}
	}
	if mapConfig == nil {
		return d.warnResponse("Map with ID "+mapId+" is not currently installed. No action taken.", "map_id", mapId)
	}

	mapDataPath := path.Join(d.getMapDataPath(), mapConfig.Code)
	if err := os.RemoveAll(mapDataPath); err != nil {
		return d.throwError("Failed to remove map data files", err, "map_id", mapId)
	}
	tilePath := path.Join(d.getMapTilePath(), mapConfig.Code+".pmtiles")
	if err := os.Remove(tilePath); err != nil {
		return d.throwError("Failed to remove map tile files", err, "map_id", mapId)
	}
	os.Remove(path.Join(d.getMapThumbnailPath(), mapConfig.Code+".svg")) // Doesn't matter if this fails, thumbnail is optional and may not exist
	d.Registry.RemoveInstalledMap(mapId)
	return d.successResponse("Map uninstalled successfully", "map_id", mapId)
}

// InstallMod handles the installation of a mod given its ID and version, including downloading, extracting, and updating the registry.
func (d *Downloader) InstallMod(modId string, version string) types.GenericResponse {
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.throwErrorSimple("Cannot install mod because app config paths are not properly configured. " +
			"Please set valid paths in the config before installing mods.")
	}
	modInfo, err := d.Registry.GetMod(modId)
	if err != nil {
		return d.throwError("Failed to get mod info from registry", err, "mod_id", modId)
	}

	versions, err := d.Registry.GetVersions(modInfo.Update.Type, modInfo.Update.URL)
	if err != nil {
		return d.throwError("Failed to get mod versions from registry", err, "mod_id", modId)
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return d.throwErrorSimple("Specified version not found for mod", "mod_id", modId, "version", version)
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.throwErrorSimple("Failed to download mod zip: "+downloadResp.Message, "mod_id", modId, "version", version)
	}

	extractResp := d.handleModExtract(downloadResp.Path, modId)
	if extractResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.throwErrorSimple("Failed to extract mod zip: "+extractResp.Message, "mod_id", modId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMod(modId, version)
	return d.successResponse("Mod installed successfully", "mod_id", modId, "version", version)
}

// InstallMap handles the installation of a map given its ID and version, including downloading, extracting, validating files, and updating the registry.
func (d *Downloader) InstallMap(mapId string, version string) types.MapExtractResponse {
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.throwMapExtractErrorSimple("Invalid configuration", "map_id", mapId, "version", version)
	}
	mapInfo, err := d.Registry.GetMap(mapId)
	if err != nil {
		return d.throwMapExtractError("Failed to get map info from registry", err, "map_id", mapId)
	}

	versions, err := d.Registry.GetVersions(mapInfo.Update.Type, mapInfo.Update.URL)
	if err != nil {
		return d.throwMapExtractError("Failed to get map versions from registry", err, "map_id", mapId)
	}

	var versionInfo *types.VersionInfo = nil
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return d.throwMapExtractErrorSimple("Specified version not found for map", "map_id", mapId, "version", version)
	}

	downloadResp := d.downloadTempZip(versionInfo.DownloadURL)
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.throwMapExtractErrorSimple("Failed to download map zip: "+downloadResp.Message, "map_id", mapId, "version", version)
	}

	extractResp := d.handleMapExtract(downloadResp.Path)
	if extractResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.throwMapExtractErrorSimple("Failed to extract map zip: "+extractResp.Message, "map_id", mapId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMap(mapId, version, extractResp.Config)
	return extractResp
}

// downloadTempZip downloads a zip file from the given URL and saves it to a temporary location, returning the path or an error message.
func (d *Downloader) downloadTempZip(url string) types.DownloadTempResponse {
	if err := os.MkdirAll(d.tempPath, os.ModePerm); err != nil {
		return d.throwDownloadError("Failed to create temp directory", err, "url", url)
	}

	file, err := os.CreateTemp(d.tempPath, "download-*.zip")
	if err != nil {
		return d.throwDownloadError("Failed to create temp file", err, "url", url)
	}
	defer file.Close()
	zip, err := http.Get(url)

	if err != nil {
		return d.throwDownloadError("Failed to download file", err, "url", url)
	}
	defer zip.Body.Close()

	if zip.StatusCode != http.StatusOK {
		return d.throwDownloadErrorSimple("Failed to download file: unexpected status code", "url", url, "status_code", zip.StatusCode)
	}

	_, err = io.Copy(file, zip.Body)
	if err != nil {
		return d.throwDownloadError("Failed to save file", err, "url", url)
	}

	return d.successDownloadResponse("File downloaded successfully", file.Name(), "url", url)
}

// getVanillaMapCodes returns the city codes of maps included with the game.
func (d *Downloader) getVanillaMapCodes() []string {
	cfgResult := d.Config.GetConfig()
	if !cfgResult.Validation.IsValid() {
		log.Printf("Warning: Invalid Config: %v", cfgResult.Validation)
		return []string{}
	}
	reader, err := os.Open(path.Join(cfgResult.Config.MetroMakerDataPath, "cities", "latest-cities.yml"))
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

// Used by handleMapExtract to check for vanilla/duplicate map codes
func (d *Downloader) isMapCodeTaken(code string) bool {
	return slices.Contains(d.getVanillaMapCodes(), code) || slices.Contains(d.Registry.GetInstalledMapCodes(), code)
}

// handleModExtract processes the downloaded mod zip file, extracts it to the appropriate location, and returns a success or error message.
func (d *Downloader) handleModExtract(filePath string, modId string) types.GenericResponse {
	return extractMod(d, filePath, modId)
}

// handleMapExtract processes the downloaded map zip file, validates required files, extracts them to the appropriate locations, and returns the map config or an error message.
func (d *Downloader) handleMapExtract(filePath string) types.MapExtractResponse {
	return extractMap(d, filePath)
}

func requiredFilesPresent(filesFound map[string]types.FileFoundStruct) bool {
	for _, fileStruct := range filesFound {
		if fileStruct.Required && !fileStruct.Found {
			return false
		}
	}
	return true
}

