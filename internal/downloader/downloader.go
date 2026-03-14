package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"railyard/internal/config"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/types"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"go.yaml.in/yaml/v4"
)

type ExtractProgressFunc func(itemId string, extracted int64, total int64)

var isGitHubDownloadHost = func(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(h, "github.com") || strings.Contains(h, "githubusercontent.com")
}

var downloaderHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   types.RequestTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   types.RequestTimeout,
		ResponseHeaderTimeout: types.RequestTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

type Downloader struct {
	tempPath          string
	mapTilePath       string
	Registry          *registry.Registry
	Config            *config.Config
	Logger            logger.Logger
	OnProgress        types.ProgressFunc
	OnExtractProgress ExtractProgressFunc

	downloadMu   sync.Mutex
	downloadCond *sync.Cond
	queue        []*downloadOperation
	// Track pending operations so that they may be superseded by a newer request for the same asset
	pending map[downloadQueueKey]*downloadOperation
	// Track running operations to prevent concurrent operations for the same asset
	running map[downloadQueueKey]*downloadOperation
	ctx     context.Context
}

// downloadQueueKey is used to coalesce download operations for a specific asset (mod/map) ensuring that only one operation is in process or queued at a given time
type downloadQueueKey struct {
	assetType types.AssetType
	assetID   string
}

type downloadOperation struct {
	key              string
	assetKey         downloadQueueKey
	run              func() operationResult
	supersededResult operationResult
	completed        chan operationResult
}

type operationResult struct {
	genericResponse        types.GenericResponse
	assetInstallResponse   types.AssetInstallResponse
	assetUninstallResponse types.AssetUninstallResponse
}

// operationAction is an internal type to categorize all possible download actions within the queue
type operationAction string

const (
	operationActionInstall   operationAction = "install"
	operationActionUninstall operationAction = "uninstall"
)

func isValidOperationAction(action operationAction) bool {
	switch action {
	case operationActionInstall, operationActionUninstall:
		return true
	default:
		return false
	}
}

// NewDownloader creates a new Downloader instance with necessary paths and references.
func NewDownloader(cfg *config.Config, reg *registry.Registry, l logger.Logger) *Downloader {
	d := &Downloader{
		mapTilePath: path.Join(paths.AppDataRoot(), "tiles"),
		tempPath:    path.Join(paths.AppDataRoot(), "temp"),
		Registry:    reg,
		Config:      cfg,
		Logger:      l,
	}
	d.startQueue()
	return d
}

func (d *Downloader) SetContext(ctx context.Context) {
	d.ctx = ctx
}

// startQueue initializes the download queue and starts the worker goroutine if it hasn't been started yet.
func (d *Downloader) startQueue() {
	d.downloadMu.Lock()
	defer d.downloadMu.Unlock()
	// Ensure the queue is started only once
	if d.downloadCond != nil {
		return
	}
	d.downloadCond = sync.NewCond(&d.downloadMu)
	d.pending = make(map[downloadQueueKey]*downloadOperation)
	d.running = make(map[downloadQueueKey]*downloadOperation)
	go d.runQueue()
}

// runQueue processes download operations sequentially, ensuring that only one operation is present in the queue at a time.
func (d *Downloader) runQueue() {
	for {
		// Lock the queue and wait for an operation to be added if the queue is empty
		d.downloadMu.Lock()
		for len(d.queue) == 0 {
			d.downloadCond.Wait()
		}
		op := d.queue[0]
		d.queue = d.queue[1:]
		if pending, ok := d.pending[op.assetKey]; ok && pending == op {
			delete(d.pending, op.assetKey)
		}
		d.running[op.assetKey] = op
		// Unlock to allow other operations to be enqueued during runs
		d.downloadMu.Unlock()

		result := op.run()

		// Lock the queue again to perform state mutation
		d.downloadMu.Lock()
		if running, ok := d.running[op.assetKey]; ok && running == op {
			delete(d.running, op.assetKey)
		}
		d.downloadMu.Unlock()

		op.completed <- result
		close(op.completed)
		if d.ctx != nil {
			wailsruntime.EventsEmit(d.ctx, "registry:update") // Emit update event after each operation completes to trigger UI refresh
		}
	}
}

// replaceQueuedOperation replaces an existing queued operation for the same asset with a new operation, returning a boolean to indicate success
func (d *Downloader) replaceQueuedOperation(target *downloadOperation, replacement *downloadOperation) bool {
	for i, queued := range d.queue {
		if queued == target {
			d.queue[i] = replacement
			return true
		}
	}
	return false
}

// enqueueOperation adds a new operation to the queue using asset-level coalescing.
// Only one queued operation per asset is retained; later requests supersede older pending requests.
func (d *Downloader) enqueueOperation(assetKey downloadQueueKey, key string, run func() operationResult, supersededResult operationResult) operationResult {
	d.startQueue()

	op := &downloadOperation{
		key:              key,
		assetKey:         assetKey,
		run:              run,
		supersededResult: supersededResult,
		completed:        make(chan operationResult, 1),
	}

	d.downloadMu.Lock()
	// If there's an existing pending operation for the same asset, replace it in-place in the queue and mark it as superseded.
	// Prefer in-place replacement so that any other callers waiting on the initial result no longer have to wait until all other pending requests are completed
	if existingPending, ok := d.pending[assetKey]; ok {
		replaced := d.replaceQueuedOperation(existingPending, op)
		if !replaced {
			// Fallback guard: if pending bookkeeping is ever out of sync, keep progress by appending.
			d.queue = append(d.queue, op)
		}
		existingPending.completed <- existingPending.supersededResult
		close(existingPending.completed)
		d.pending[assetKey] = op
		d.downloadCond.Signal()
		d.downloadMu.Unlock()
		return <-op.completed
	}

	d.queue = append(d.queue, op)
	d.pending[assetKey] = op
	d.downloadCond.Signal()
	d.downloadMu.Unlock()

	return <-op.completed
}

// operationKey generates a unique key for a given operation based on its action, asset type, asset ID, and version.
func (d *Downloader) operationKey(action operationAction, assetType types.AssetType, assetID string, version string) string {
	if !isValidOperationAction(action) {
		// Hard panic here as this is an issue with implementation
		panic(fmt.Sprintf("invalid downloader operation action: %q", action))
	}

	return strings.Join([]string{
		string(action),
		string(assetType),
		assetID,
		version,
	}, "|")
}

// getModPath returns the filesystem path for installed mods.
func (d *Downloader) getModPath() string {
	return path.Join(d.Config.Cfg.MetroMakerDataPath, "mods")
}

func (d *Downloader) logStatus(status types.Status, message string, attrs ...any) types.GenericResponse {
	response := types.GenericResponse{Status: status, Message: message}
	if d.Logger != nil {
		d.Logger.LogResponse("Downloader response", response, attrs...)
	}
	return response
}

func (d *Downloader) successResponse(message string, attrs ...any) types.GenericResponse {
	return d.logStatus(types.ResponseSuccess, message, attrs...)
}

func (d *Downloader) warnResponse(message string, attrs ...any) types.GenericResponse {
	return d.logStatus(types.ResponseWarn, message, attrs...)
}

func withError(message string, err error) string {
	if err == nil {
		return message
	}
	return message + ": " + err.Error()
}

func (d *Downloader) throwError(message string, err error, attrs ...any) types.GenericResponse {
	return d.logStatus(types.ResponseError, withError(message, err), attrs...)
}

func (d *Downloader) throwDownloadError(message string, err error, attrs ...any) types.DownloadTempResponse {
	return d.toDownloadResponse(d.throwError(message, err, attrs...), "")
}

func (d *Downloader) toDownloadResponse(base types.GenericResponse, path string) types.DownloadTempResponse {
	return types.DownloadTempResponse{
		GenericResponse: base,
		Path:            path,
	}
}

func (d *Downloader) installResponse(assetType types.AssetType, assetID string, version string, config types.ConfigData, base types.GenericResponse, errorCode types.DownloaderErrorType) types.AssetInstallResponse {
	return types.AssetInstallResponse{
		GenericResponse: base,
		AssetType:       assetType,
		AssetID:         assetID,
		Version:         version,
		Config:          config,
		ErrorType:       errorCode,
	}
}

func (d *Downloader) installSuccess(assetType types.AssetType, assetID string, version string, config types.ConfigData, message string, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.successResponse(message, attrs...), "")
}

func (d *Downloader) installWarn(assetType types.AssetType, assetID string, version string, config types.ConfigData, message string, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.warnResponse(message, attrs...), "")
}

func (d *Downloader) installError(assetType types.AssetType, assetID string, version string, config types.ConfigData, errorCode types.DownloaderErrorType, message string, err error, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.throwError(message, err, attrs...), errorCode)
}

func (d *Downloader) uninstallResponse(assetType types.AssetType, assetID string, base types.GenericResponse, errorCode types.DownloaderErrorType) types.AssetUninstallResponse {
	return types.AssetUninstallResponse{
		GenericResponse: base,
		AssetType:       assetType,
		AssetID:         assetID,
		ErrorType:       errorCode,
	}
}

func (d *Downloader) uninstallSuccess(assetType types.AssetType, assetID string, message string, attrs ...any) types.AssetUninstallResponse {
	return d.uninstallResponse(assetType, assetID, d.successResponse(message, attrs...), "")
}

func (d *Downloader) uninstallWarn(assetType types.AssetType, assetID string, errorCode types.DownloaderErrorType, message string, attrs ...any) types.AssetUninstallResponse {
	return d.uninstallResponse(assetType, assetID, d.warnResponse(message, attrs...), errorCode)
}

func (d *Downloader) uninstallError(assetType types.AssetType, assetID string, errorCode types.DownloaderErrorType, message string, err error, attrs ...any) types.AssetUninstallResponse {
	return d.uninstallResponse(assetType, assetID, d.throwError(message, err, attrs...), errorCode)
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

type installedState struct {
	version   string
	mapConfig types.ConfigData
}

var assetTypeLabels = map[types.AssetType]string{
	types.AssetTypeMap: "Map",
	types.AssetTypeMod: "Mod",
}

func (d *Downloader) getInstalledState(assetType types.AssetType, assetID string) (installedState, bool) {
	switch assetType {
	case types.AssetTypeMod:
		for _, mod := range d.Registry.GetInstalledMods() {
			if mod.ID == assetID {
				return installedState{version: mod.Version}, true
			}
		}
	case types.AssetTypeMap:
		for _, installedMap := range d.Registry.GetInstalledMaps() {
			if installedMap.ID == assetID {
				return installedState{version: installedMap.Version, mapConfig: installedMap.MapConfig}, true
			}
		}
	}
	return installedState{}, false
}

func (d *Downloader) supersededOperationResult(action operationAction, assetType types.AssetType, assetID string, version string) operationResult {
	message := "Operation superseded by newer queued request. No action taken."
	base := d.successResponse(message, "asset_type", assetType, "asset_id", assetID, "action", action, "version", version)

	if action == operationActionInstall {
		return operationResult{
			assetInstallResponse: d.installResponse(assetType, assetID, version, types.ConfigData{}, base, ""),
		}
	}
	return operationResult{
		genericResponse:        base,
		assetUninstallResponse: d.uninstallResponse(assetType, assetID, base, ""),
	}
}

// UninstallAsset handles uninstallation for all supported asset types.
func (d *Downloader) UninstallAsset(assetType types.AssetType, assetID string) types.AssetUninstallResponse {
	if !types.IsValidAssetType(assetType) {
		return d.uninstallError(
			assetType,
			assetID,
			types.UninstallErrorInvalidAssetType,
			"Invalid asset type",
			nil,
			"asset_type", assetType, "asset_id", assetID,
		)
	}

	key := d.operationKey(operationActionUninstall, assetType, assetID, "")
	assetKey := downloadQueueKey{assetType: assetType, assetID: assetID}
	result := d.enqueueOperation(assetKey, key, func() operationResult {
		switch assetType {
		case types.AssetTypeMap:
			return operationResult{assetUninstallResponse: d.uninstallMapNow(assetID)}
		case types.AssetTypeMod:
			return operationResult{assetUninstallResponse: d.uninstallModNow(assetID)}
		default:
			return operationResult{assetUninstallResponse: d.uninstallError(
				assetType,
				assetID,
				types.UninstallErrorInvalidAssetType,
				"Invalid asset type",
				nil,
				"asset_type", assetType, "asset_id", assetID,
			)}
		}
	}, d.supersededOperationResult(operationActionUninstall, assetType, assetID, ""))
	return result.assetUninstallResponse
}

func (d *Downloader) uninstallModNow(modId string) types.AssetUninstallResponse {
	if _, ok := d.getInstalledState(types.AssetTypeMod, modId); !ok {
		return d.uninstallWarn(
			types.AssetTypeMod,
			modId,
			types.UninstallErrorNotInstalled,
			fmt.Sprintf("%s with ID %s is not currently installed. No action taken.", assetTypeLabels[types.AssetTypeMod], modId),
			"asset_type", types.AssetTypeMod,
			"asset_id", modId,
		)
	}
	modPath := path.Join(d.getModPath(), modId)
	if err := os.RemoveAll(modPath); err != nil {
		return d.uninstallError(types.AssetTypeMod, modId, types.UninstallErrorFilesystem, "Failed to remove mod files", err, "mod_id", modId)
	}
	d.Registry.RemoveInstalledMod(modId)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after uninstalling mod", "error", err)
	}
	return d.uninstallSuccess(types.AssetTypeMod, modId, "Mod uninstalled successfully", "mod_id", modId)
}

func (d *Downloader) uninstallMapNow(mapId string) types.AssetUninstallResponse {
	installedMap, ok := d.getInstalledState(types.AssetTypeMap, mapId)
	if !ok {
		return d.uninstallWarn(
			types.AssetTypeMap,
			mapId,
			types.UninstallErrorNotInstalled,
			fmt.Sprintf("%s with ID %s is not currently installed. No action taken.", assetTypeLabels[types.AssetTypeMap], mapId),
			"asset_type", types.AssetTypeMap,
			"asset_id", mapId,
		)
	}
	mapConfig := installedMap.mapConfig

	mapDataPath := path.Join(d.getMapDataPath(), mapConfig.Code)
	if err := os.RemoveAll(mapDataPath); err != nil {
		return d.uninstallError(types.AssetTypeMap, mapId, types.UninstallErrorFilesystem, "Failed to remove map data files", err, "map_id", mapId)
	}
	tilePath := path.Join(d.getMapTilePath(), mapConfig.Code+".pmtiles")
	if err := os.Remove(tilePath); err != nil {
		return d.uninstallError(types.AssetTypeMap, mapId, types.UninstallErrorFilesystem, "Failed to remove map tile files", err, "map_id", mapId)
	}
	os.Remove(path.Join(d.getMapThumbnailPath(), mapConfig.Code+".svg")) // Doesn't matter if this fails, thumbnail is optional and may not exist
	d.Registry.RemoveInstalledMap(mapId)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after uninstalling map", "error", err)
	}
	return d.uninstallSuccess(types.AssetTypeMap, mapId, "Map uninstalled successfully", "map_id", mapId)
}

// InstallAsset handles installation for all supported asset types.
func (d *Downloader) InstallAsset(assetType types.AssetType, assetID string, version string) types.AssetInstallResponse {
	if !types.IsValidAssetType(assetType) {
		return d.installError(
			assetType,
			assetID,
			version,
			types.ConfigData{},
			types.InstallErrorInvalidAssetType,
			"Invalid asset type",
			nil,
			"asset_type", assetType, "asset_id", assetID, "version", version,
		)
	}

	key := d.operationKey(operationActionInstall, assetType, assetID, version)
	assetKey := downloadQueueKey{assetType: assetType, assetID: assetID}
	result := d.enqueueOperation(assetKey, key, func() operationResult {
		switch assetType {
		case types.AssetTypeMap:
			return operationResult{assetInstallResponse: d.installMapNow(assetID, version)}
		case types.AssetTypeMod:
			return operationResult{assetInstallResponse: d.installModNow(assetID, version)}
		default:
			return operationResult{assetInstallResponse: d.installError(
				assetType,
				assetID,
				version,
				types.ConfigData{},
				types.InstallErrorInvalidAssetType,
				"Invalid asset type",
				nil,
				"asset_type", assetType, "asset_id", assetID, "version", version,
			)}
		}
	}, d.supersededOperationResult(operationActionInstall, assetType, assetID, version))
	return result.assetInstallResponse
}

// installModNow handles the installation of a mod given its ID and version, including downloading, extracting, and updating the registry.
func (d *Downloader) installModNow(modId string, version string) types.AssetInstallResponse {
	d.Logger.Info("InstallMod started", "mod_id", modId, "version", version)
	if state, installed := d.getInstalledState(types.AssetTypeMod, modId); installed && state.version == version {
		return d.installWarn(
			types.AssetTypeMod,
			modId,
			version,
			types.ConfigData{},
			fmt.Sprintf("%s already installed at requested version. No action taken.", assetTypeLabels[types.AssetTypeMod]),
			"asset_type", types.AssetTypeMod,
			"asset_id", modId,
			"version", version,
		)
	}
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.installError(
			types.AssetTypeMod,
			modId,
			version,
			types.ConfigData{},
			types.InstallErrorInvalidConfig,
			"Cannot install mod because app config paths are not properly configured. Please set valid paths in the config before installing mods.",
			nil,
		)
	}
	modInfo, err := d.Registry.GetMod(modId)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorRegistryLookup, "Failed to get mod info from registry", err, "mod_id", modId)
	}

	source := modInfo.Update.URL
	if modInfo.Update.Type == "github" {
		source = modInfo.Update.Repo
	}
	d.Logger.Info("Fetching available versions", "mod_id", modId, "update_type", modInfo.Update.Type, "source", source)
	versions, err := d.Registry.GetVersions(modInfo.Update.Type, source)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorVersionLookup, "Failed to get mod versions from registry", err, "mod_id", modId)
	}

	availableVersions := make([]string, len(versions))
	for i, v := range versions {
		availableVersions[i] = v.Version
	}
	d.Logger.Info("Fetched available versions", "mod_id", modId, "requested_version", version, "available_versions", availableVersions)

	var versionInfo *types.VersionInfo
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorVersionNotFound, "Specified version not found for mod", nil, "mod_id", modId, "version", version, "available_versions", availableVersions)
	}

	d.Logger.Info("Downloading mod", "mod_id", modId, "version", version, "download_url", versionInfo.DownloadURL)
	downloadResp := d.downloadTempZip(versionInfo.DownloadURL, modId)
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorDownloadFailed, "Failed to download mod zip: "+downloadResp.Message, nil, "mod_id", modId, "version", version)
	}

	if err := d.verifySHA256(downloadResp.Path, versionInfo.SHA256); err != nil {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorChecksumFailed, "SHA-256 integrity check failed", err, "mod_id", modId, "version", version)
	}

	d.Logger.Info("Extracting mod", "mod_id", modId, "version", version, "temp_path", downloadResp.Path)
	extractResp := d.handleModExtract(downloadResp.Path, modId, version)
	if extractResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, extractResp.ErrorType, "Failed to extract mod zip: "+extractResp.Message, nil, "mod_id", modId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMod(modId, version)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after installing mod", "error", err)
	}
	d.Logger.Info("InstallMod completed", "mod_id", modId, "version", version)
	return d.installSuccess(types.AssetTypeMod, modId, version, types.ConfigData{}, "Mod installed successfully", "mod_id", modId, "version", version)
}

// installMapNow handles the installation of a map given its ID and version, including downloading, extracting, validating files, and updating the registry.
func (d *Downloader) installMapNow(mapId string, version string) types.AssetInstallResponse {
	d.Logger.Info("InstallMap started", "map_id", mapId, "version", version)
	if state, installed := d.getInstalledState(types.AssetTypeMap, mapId); installed && state.version == version {
		return d.installWarn(
			types.AssetTypeMap,
			mapId,
			version,
			state.mapConfig,
			fmt.Sprintf("%s already installed at requested version. No action taken.", assetTypeLabels[types.AssetTypeMap]),
			"asset_type", types.AssetTypeMap,
			"asset_id", mapId,
			"version", version,
		)
	}
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorInvalidConfig, "Invalid configuration", nil, "map_id", mapId, "version", version)
	}
	mapInfo, err := d.Registry.GetMap(mapId)
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorRegistryLookup, "Failed to get map info from registry", err, "map_id", mapId)
	}

	source := mapInfo.Update.URL
	if mapInfo.Update.Type == "github" {
		source = mapInfo.Update.Repo
	}
	d.Logger.Info("Fetching available versions", "map_id", mapId, "update_type", mapInfo.Update.Type, "source", source)
	versions, err := d.Registry.GetVersions(mapInfo.Update.Type, source)
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorVersionLookup, "Failed to get map versions from registry", err, "map_id", mapId)
	}

	availableVersions := make([]string, len(versions))
	for i, v := range versions {
		availableVersions[i] = v.Version
	}
	d.Logger.Info("Fetched available versions", "map_id", mapId, "requested_version", version, "available_versions", availableVersions)

	var versionInfo *types.VersionInfo
	for _, v := range versions {
		if v.Version == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorVersionNotFound, "Specified version not found for map", nil, "map_id", mapId, "version", version, "available_versions", availableVersions)
	}

	d.Logger.Info("Downloading map", "map_id", mapId, "version", version, "download_url", versionInfo.DownloadURL)
	downloadResp := d.downloadTempZip(versionInfo.DownloadURL, mapId)
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorDownloadFailed, "Failed to download map zip: "+downloadResp.Message, nil, "map_id", mapId, "version", version)
	}

	if err := d.verifySHA256(downloadResp.Path, versionInfo.SHA256); err != nil {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorChecksumFailed, "SHA-256 integrity check failed", err, "map_id", mapId, "version", version)
	}

	d.Logger.Info("Extracting map", "map_id", mapId, "version", version, "temp_path", downloadResp.Path)
	extractResp := d.handleMapExtract(downloadResp.Path, mapId, version)
	if extractResp.Status == types.ResponseError {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, extractResp.ErrorType, "Failed to extract map zip: "+extractResp.Message, nil, "map_id", mapId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMap(mapId, version, extractResp.Config)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after installing map", "error", err)
	}
	d.Logger.Info("InstallMap completed", "map_id", mapId, "version", version)
	return extractResp
}

// downloadTempZip downloads a zip file from the given URL and saves it to a temporary location, returning the path or an error message.
func (d *Downloader) downloadTempZip(url string, itemId string) types.DownloadTempResponse {
	if err := os.MkdirAll(d.tempPath, os.ModePerm); err != nil {
		return d.throwDownloadError("Failed to create temp directory", err, "url", url)
	}

	file, err := os.CreateTemp(d.tempPath, "download-*.zip")
	if err != nil {
		return d.throwDownloadError("Failed to create temp file", err, "url", url)
	}
	defer file.Close()

	zip, err := d.doDownloadRequest(url, d.Config.GetGithubToken())

	if err != nil {
		return d.throwDownloadError("Failed to download file", err, "url", url)
	}
	defer zip.Body.Close()

	if zip.StatusCode != http.StatusOK {
		return d.throwDownloadError("Failed to download file: unexpected status code", nil, "url", url, "status_code", zip.StatusCode)
	}

	var reader io.Reader = zip.Body
	if d.OnProgress != nil {
		reader = &types.ProgressReader{
			Reader:     zip.Body,
			Total:      zip.ContentLength,
			ItemId:     itemId,
			OnProgress: d.OnProgress,
		}
	}

	_, err = io.Copy(file, reader)
	if err != nil {
		return d.throwDownloadError("Failed to save file", err, "url", url)
	}

	return d.toDownloadResponse(d.successResponse("File downloaded successfully", "url", url), file.Name())
}

func (d *Downloader) doDownloadRequest(downloadURL, githubToken string) (*http.Response, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Railyard-Desktop-App")
	trimmedToken := strings.TrimSpace(githubToken)
	tokenApplied := false
	if trimmedToken != "" && shouldAuthenticateDownloadURL(downloadURL) {
		req.Header.Set("Authorization", "Bearer "+trimmedToken)
		tokenApplied = true
	}

	resp, err := downloaderHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if tokenApplied && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
		resp.Body.Close()
		reqNoAuth, reqErr := http.NewRequest("GET", downloadURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		reqNoAuth.Header.Set("User-Agent", "Railyard-Desktop-App")
		return downloaderHTTPClient.Do(reqNoAuth)
	}

	return resp, nil
}

func shouldAuthenticateDownloadURL(downloadURL string) bool {
	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return false
	}
	return isGitHubDownloadHost(parsed.Hostname())
}

// verifySHA256 checks the SHA-256 hash of a downloaded file against an expected hash.
// If expectedHash is empty, the check is skipped (GitHub releases rely on GitHub's own integrity).
func (d *Downloader) verifySHA256(filePath string, expectedHash string) error {
	if expectedHash == "" {
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for hash verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to compute SHA-256: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("expected %s, got %s", expectedHash, actual)
	}
	return nil
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
func (d *Downloader) handleModExtract(filePath string, modId string, version string) types.AssetInstallResponse {
	return extractMod(d, filePath, modId, version)
}

// handleMapExtract processes the downloaded map zip file, validates required files, extracts them to the appropriate locations, and returns the map config or an error message.
func (d *Downloader) handleMapExtract(filePath string, mapId string, version string) types.AssetInstallResponse {
	return extractMap(d, filePath, mapId, version)
}

func requiredFilesPresent(filesFound map[string]types.FileFoundStruct) bool {
	for _, fileStruct := range filesFound {
		if fileStruct.Required && !fileStruct.Found {
			return false
		}
	}
	return true
}
