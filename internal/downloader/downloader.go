package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"

	"railyard/internal/config"
	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/requests"
	"railyard/internal/types"

	semver "github.com/Masterminds/semver/v3"
	"go.yaml.in/yaml/v4"
)

type ExtractProgressFunc func(itemId string, extracted int64, total int64)
type CancelledFunc func(itemID string, assetType types.AssetType, phase string)
type DependencyInstalledFunc func(modID string, itemType types.AssetType, version types.Version)
type RegistryUpdateFunc func()
type GameVersionFunc func() types.GameVersionResponse

// TODO: Consider adding this as an injectable dependency for other services
var downloaderHTTPClient = requests.NewDownloadClient()

type Downloader struct {
	tempPath          string
	mapTilePath       string
	Registry          *registry.Registry
	Config            *config.Config
	Logger            logger.Logger
	OnProgress        types.ProgressFunc
	OnExtractProgress ExtractProgressFunc
	OnCancelled       CancelledFunc
	OnRegistryUpdate  RegistryUpdateFunc
	InstallDependency DependencyInstalledFunc
	GetGameVersion    GameVersionFunc

	downloadMu   sync.Mutex
	downloadCond *sync.Cond
	queue        []*downloadOperation
	// Track pending operations so that they may be superseded by a newer request for the same asset
	pending map[downloadQueueKey]*downloadOperation
	// Track running operations to prevent concurrent operations for the same asset
	running map[downloadQueueKey]*downloadOperation
}

// downloadQueueKey is used to coalesce download operations for a specific asset (mod/map) ensuring that only one operation is in process or queued at a given time
type downloadQueueKey struct {
	assetType types.AssetType
	assetID   string
}

type downloadOperation struct {
	action           operationAction
	key              string
	assetKey         downloadQueueKey
	run              func() operationResult
	supersededResult operationResult
	cancel           context.CancelFunc
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

const (
	cancelledPhaseQueued  = "queued"
	cancelledPhaseRunning = "running"
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
		mapTilePath: paths.JoinLocalPath(paths.AppDataRoot(), "tiles"),
		tempPath:    paths.JoinLocalPath(paths.AppDataRoot(), "temp"),
		Registry:    reg,
		Config:      cfg,
		Logger:      l,
	}
	d.startQueue()
	return d
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
		if d.OnRegistryUpdate != nil {
			d.OnRegistryUpdate() // Trigger UI refresh after each operation completes.
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

// removeQueuedOperation removes a queued operation from the queue, returning a boolean to indicate success
func (d *Downloader) removeQueuedOperation(target *downloadOperation) bool {
	for i, queued := range d.queue {
		if queued != target {
			continue
		}
		d.queue = append(d.queue[:i], d.queue[i+1:]...)
		return true
	}
	return false
}

// cancelPendingQueuedInstall removes a queued install for the same asset when an uninstall arrives, but only when that asset is not already installed
func (d *Downloader) cancelPendingQueuedInstall(assetType types.AssetType, assetID string, assetKey downloadQueueKey) bool {
	if _, installed := d.getInstalledState(assetType, assetID); installed {
		return false
	}

	d.downloadMu.Lock()
	defer d.downloadMu.Unlock()

	pending, ok := d.pending[assetKey]
	if !ok || pending.action != operationActionInstall {
		return false
	}
	if !d.removeQueuedOperation(pending) {
		return false
	}

	delete(d.pending, assetKey)
	if pending.cancel != nil {
		pending.cancel()
	}
	pending.completed <- pending.supersededResult
	close(pending.completed)
	d.OnCancelled(assetID, assetType, cancelledPhaseQueued)

	return true
}

// enqueueOperation adds a new operation to the queue using asset-level coalescing.
// Only one queued operation per asset is retained; later requests supersede older pending requests.
func (d *Downloader) enqueueOperation(action operationAction, assetKey downloadQueueKey, key string, run func() operationResult, supersededResult operationResult, cancel context.CancelFunc) operationResult {
	d.startQueue()

	op := &downloadOperation{
		action:           action,
		key:              key,
		assetKey:         assetKey,
		run:              run,
		supersededResult: supersededResult,
		cancel:           cancel,
		completed:        make(chan operationResult, 1),
	}

	d.downloadMu.Lock()
	if action == operationActionUninstall {
		// If an uninstall action is enqueued while an install is running for the same asset, attempt to cancel the install
		if running, ok := d.running[assetKey]; ok && running.action == operationActionInstall && running.cancel != nil {
			running.cancel()
			d.OnCancelled(assetKey.assetID, assetKey.assetType, cancelledPhaseRunning)
		}
	}
	// If there's an existing pending operation for the same asset, replace it in-place in the queue and mark it as superseded.
	// Prefer in-place replacement so that any other callers waiting on the initial result no longer have to wait until all other pending requests are completed
	if existingPending, ok := d.pending[assetKey]; ok {
		replaced := d.replaceQueuedOperation(existingPending, op)
		if !replaced {
			// Fallback guard: if pending bookkeeping is ever out of sync, keep progress by appending.
			d.queue = append(d.queue, op)
		}
		// Cancel the context of the existing pending operation (e.g. if it's currently running an install) so that it can exit early and return the superseded result.
		if existingPending.cancel != nil {
			existingPending.cancel()
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
	return paths.JoinLocalPath(d.Config.Cfg.MetroMakerDataPath, "mods")
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

func (d *Downloader) installResponse(
	assetType types.AssetType,
	assetID string,
	version string,
	config types.ConfigData,
	base types.GenericResponse,
	errorCode types.DownloaderErrorType,
	mapCodeConflict *types.MapCodeConflict,
) types.AssetInstallResponse {
	return types.AssetInstallResponse{
		GenericResponse: base,
		AssetType:       assetType,
		AssetID:         assetID,
		Version:         version,
		Config:          config,
		ErrorType:       errorCode,
		MapCodeConflict: mapCodeConflict,
	}
}

func (d *Downloader) installSuccess(assetType types.AssetType, assetID string, version string, config types.ConfigData, message string, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.successResponse(message, attrs...), "", nil)
}

func (d *Downloader) installWarn(assetType types.AssetType, assetID string, version string, config types.ConfigData, mapCodeConflict *types.MapCodeConflict, message string, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.warnResponse(message, attrs...), "", mapCodeConflict)
}

func (d *Downloader) installError(assetType types.AssetType, assetID string, version string, config types.ConfigData, errorCode types.DownloaderErrorType, message string, err error, attrs ...any) types.AssetInstallResponse {
	return d.installResponse(assetType, assetID, version, config, d.throwError(message, err, attrs...), errorCode, nil)
}

func (d *Downloader) uninstallResponse(assetType types.AssetType, assetID string, base types.GenericResponse, errorCode types.DownloaderErrorType) types.AssetUninstallResponse {
	return types.AssetUninstallResponse{
		GenericResponse: base,
		AssetType:       assetType,
		AssetID:         assetID,
		ErrorType:       errorCode,
	}
}

func (d *Downloader) installConfigError(assetType types.AssetType, assetID string, version string) types.AssetInstallResponse {
	return d.installError(
		types.AssetTypeMod,
		assetID,
		version,
		types.ConfigData{},
		types.InstallErrorInvalidConfig,
		"Cannot install "+string(assetType)+" because app config paths are not properly configured. Please set valid paths in the config before installing.",
		nil,
	)
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
	return paths.JoinLocalPath(d.Config.Cfg.MetroMakerDataPath, "cities", "data")
}

// getMapTilePath returns the filesystem path for installed map tiles.
func (d *Downloader) getMapTilePath() string {
	return paths.JoinLocalPath(paths.AppDataRoot(), "tiles")
}

// getMapThumbnailPath returns the filesystem path for installed map thumbnails.
func (d *Downloader) getMapThumbnailPath() string {
	return paths.JoinLocalPath(d.Config.Cfg.MetroMakerDataPath, "public", "data", "city-maps")
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
			assetInstallResponse: d.installResponse(assetType, assetID, version, types.ConfigData{}, base, "", nil),
		}
	}
	return operationResult{
		genericResponse:        base,
		assetUninstallResponse: d.uninstallResponse(assetType, assetID, base, ""),
	}
}

// FindMapCodeConflict resolves whether the provided map city code would collide with an already-installed (local/remote) or vanilla map.
func (d *Downloader) FindMapCodeConflict(targetAssetID string, cityCode string, ignoreTargetAsset bool) (*types.MapCodeConflict, bool) {
	if cityCode == "" {
		return nil, false
	}

	for _, installedMap := range d.Registry.GetInstalledMaps() {
		// During normal install flow, ignore the target asset itself when checking for conflicts to allow for updates
		if ignoreTargetAsset && installedMap.ID == targetAssetID {
			continue
		}
		// Check for map code conflict with installed maps based on the map code in the config
		if installedMap.MapConfig.Code != cityCode {
			continue
		}

		return &types.MapCodeConflict{
			AssetConflict: types.AssetConflict{
				ExistingAssetID:   installedMap.ID,
				ExistingAssetType: types.AssetTypeMap,
				ExistingVersion:   installedMap.Version,
				ExistingIsLocal:   installedMap.IsLocal,
			},
			CityCode: cityCode,
		}, true
	}

	// Vanilla map code conflicts are determined based on the known list of vanilla map codes from latest-cities.yml.
	// Vanilla map codes will always conflict regardless of the type of install.
	for _, vanillaCode := range d.getVanillaMapCodes() {
		if vanillaCode != cityCode {
			continue
		}

		return &types.MapCodeConflict{
			AssetConflict: types.AssetConflict{
				ExistingAssetID:   "vanilla:" + cityCode,
				ExistingAssetType: types.AssetTypeMap,
				ExistingVersion:   "",
				ExistingIsLocal:   false,
			},
			CityCode: cityCode,
		}, true
	}

	return nil, false
}

func (d *Downloader) resolveMapInstallConflict(
	mapID string,
	version string,
	config types.ConfigData,
	replaceOnConflict bool,
	ignoreTargetAsset bool,
	conflictWarnMessage string,
	conflictUninstallErrorMessage string,
	baseLogAttrs ...any,
) (*types.MapCodeConflict, types.AssetInstallResponse, bool) {
	conflict, hasConflict := d.FindMapCodeConflict(mapID, config.Code, ignoreTargetAsset)
	if !hasConflict {
		return nil, types.AssetInstallResponse{}, false
	}

	warnLogAttrs := append([]any{}, baseLogAttrs...)
	warnLogAttrs = append(warnLogAttrs,
		"city_code", conflict.CityCode,
		"conflicting_asset_id", conflict.ExistingAssetID,
		"conflicting_is_local", conflict.ExistingIsLocal,
	)

	if !replaceOnConflict {
		return conflict, d.installWarn(
			types.AssetTypeMap,
			mapID,
			version,
			config,
			conflict,
			conflictWarnMessage,
			warnLogAttrs...,
		), true
	}

	if strings.HasPrefix(conflict.ExistingAssetID, "vanilla:") {
		return conflict, d.installError(
			types.AssetTypeMap,
			mapID,
			version,
			config,
			types.InstallErrorMapCodeConflict,
			"Cannot replace a vanilla map city code",
			nil,
			warnLogAttrs...,
		), true
	}

	uninstallResp := d.uninstallMapNow(conflict.ExistingAssetID)
	if uninstallResp.Status == types.ResponseError {
		errAttrs := append([]any{}, baseLogAttrs...)
		errAttrs = append(errAttrs, "conflicting_asset_id", conflict.ExistingAssetID)
		return conflict, d.installError(
			types.AssetTypeMap,
			mapID,
			version,
			config,
			types.InstallErrorMapCodeConflict,
			conflictUninstallErrorMessage,
			nil,
			errAttrs...,
		), true
	}

	return conflict, types.AssetInstallResponse{}, false
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
	if d.cancelPendingQueuedInstall(assetType, assetID, assetKey) {
		return d.uninstallWarn(
			assetType,
			assetID,
			types.UninstallErrorNotInstalled,
			"Cancelled pending install. No uninstall required.",
			"asset_type", assetType,
			"asset_id", assetID,
		)
	}
	result := d.enqueueOperation(operationActionUninstall, assetKey, key, func() operationResult {
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
	}, d.supersededOperationResult(operationActionUninstall, assetType, assetID, ""), nil)
	return result.assetUninstallResponse
}

func (d *Downloader) ImportAsset(assetType types.AssetType, zipPath string, replaceOnConflict bool) types.AssetInstallResponse {

	// TODO: Remove this and replace with isValidAssetType check once mod support is added
	if assetType != types.AssetTypeMap {
		return d.installError(
			assetType,
			"",
			"",
			types.ConfigData{},
			types.InstallErrorInvalidAssetType,
			"ImportAsset currently supports map assets only",
			nil,
			"asset_type", assetType,
		)
	}

	if !d.Config.GetConfig().Validation.IsValid() {
		return d.installConfigError(assetType, zipPath, "Local")
	}

	key := d.operationKey(operationActionInstall, assetType, zipPath, "import")
	assetKey := downloadQueueKey{assetType: assetType, assetID: zipPath}
	result := d.enqueueOperation(operationActionInstall, assetKey, key, func() operationResult {
		// TODO: Add support for local mods/other assets
		return operationResult{assetInstallResponse: d.importMapNow(zipPath, replaceOnConflict)}
	}, d.supersededOperationResult(operationActionInstall, assetType, zipPath, "import"), nil)
	return result.assetInstallResponse
}

func (d *Downloader) importMapNow(zipPath string, replaceOnConflict bool) types.AssetInstallResponse {
	configData, configErrType, configErr := files.ValidateMapArchive(zipPath)
	if configErr != nil {
		return d.installError(types.AssetTypeMap, "", "", types.ConfigData{}, configErrType, "Failed to inspect map archive", configErr, "zip_path", zipPath)
	}

	mapID := configData.Code
	version := strings.TrimSpace(configData.Version)
	if version == "" {
		version = "0.0.0"
	}

	// For local imports, detect conflicts even when the existing installed asset has the same local asset ID (e.g. same city code maps) so that local->local replacements use the standard warn-first override flow.
	conflict, conflictResp, handled := d.resolveMapInstallConflict(
		mapID,
		version,
		configData,
		replaceOnConflict,
		false,
		"Map import conflicts with an installed map. Confirm replacement to continue.",
		"Failed to remove conflicting installed map before import",
		"map_id", mapID,
		"zip_path", zipPath,
	)
	var replacedConflict *types.MapCodeConflict
	if handled {
		return conflictResp
	}
	if conflict != nil && replaceOnConflict {
		replacedConflict = conflict
	}

	extractResp := d.handleMapExtract(zipPath, mapID, version)
	if extractResp.Status == types.ResponseError {
		return extractResp
	}

	d.Registry.AddInstalledMap(mapID, version, true, extractResp.Config)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		return d.installError(types.AssetTypeMap, mapID, version, extractResp.Config, types.InstallErrorPersistStateFailed, "Failed to persist installed state after importing map", err, "map_id", mapID)
	}

	response := d.installSuccess(types.AssetTypeMap, mapID, version, extractResp.Config, "Map imported successfully", "map_id", mapID, "zip_path", zipPath)
	if extractResp.Status == types.ResponseWarn {
		response = d.installWarn(types.AssetTypeMap, mapID, version, extractResp.Config, replacedConflict, extractResp.Message, "map_id", mapID, "zip_path", zipPath)
	}
	response.MapCodeConflict = replacedConflict
	return response
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
	modPath := paths.JoinLocalPath(d.getModPath(), modId)
	if _, err := os.Stat(paths.JoinLocalPath(modPath, constants.RailyardAssetMarker)); errors.Is(err, fs.ErrNotExist) {
		return d.uninstallWarn(
			types.AssetTypeMod,
			modId,
			types.UninstallErrorNotInstalled,
			fmt.Sprintf("%s with ID %s does not appear to be installed (missing marker file). No action taken.", assetTypeLabels[types.AssetTypeMod], modId),
			"asset_type", types.AssetTypeMod,
			"asset_id", modId,
		)
	}
	if err := os.RemoveAll(modPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
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

	if _, err := os.Stat(paths.JoinLocalPath(d.getMapDataPath(), mapConfig.Code, constants.RailyardAssetMarker)); errors.Is(err, fs.ErrNotExist) {
		return d.uninstallWarn(
			types.AssetTypeMap,
			mapId,
			types.UninstallErrorNotInstalled,
			fmt.Sprintf("%s with ID %s does not appear to be installed (missing marker file). No action taken.", assetTypeLabels[types.AssetTypeMap], mapId),
			"asset_type", types.AssetTypeMap,
			"asset_id", mapId,
		)
	}

	mapDataPath := paths.JoinLocalPath(d.getMapDataPath(), mapConfig.Code)
	if err := os.RemoveAll(mapDataPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d.uninstallError(types.AssetTypeMap, mapId, types.UninstallErrorFilesystem, "Failed to remove map data files", err, "map_id", mapId)
	}
	tilePath := paths.JoinLocalPath(d.getMapTilePath(), mapConfig.Code+".pmtiles")
	if err := os.Remove(tilePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d.uninstallError(types.AssetTypeMap, mapId, types.UninstallErrorFilesystem, "Failed to remove map tile files", err, "map_id", mapId)
	}
	os.Remove(paths.JoinLocalPath(d.getMapThumbnailPath(), mapConfig.Code+".svg")) // Doesn't matter if this fails, thumbnail is optional and may not exist
	d.Registry.RemoveInstalledMap(mapId)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after uninstalling map", "error", err)
	}
	return d.uninstallSuccess(types.AssetTypeMap, mapId, "Map uninstalled successfully", "map_id", mapId)
}

// InstallAsset handles installation for all supported asset types using a structured request.
func (d *Downloader) InstallAsset(req types.InstallAssetRequest) types.AssetInstallResponse {
	if !types.IsValidAssetType(req.AssetType) {
		return d.installError(
			req.AssetType,
			req.AssetID,
			req.Version,
			types.ConfigData{},
			types.InstallErrorInvalidAssetType,
			"Invalid asset type",
			nil,
			"asset_type", req.AssetType, "asset_id", req.AssetID, "version", req.Version,
		)
	}

	replaceOnConflict := req.Map != nil && req.Map.ReplaceOnConflict
	key := d.operationKey(operationActionInstall, req.AssetType, req.AssetID, req.Version)
	assetKey := downloadQueueKey{assetType: req.AssetType, assetID: req.AssetID}
	opCtx, cancel := context.WithCancel(context.Background())
	result := d.enqueueOperation(operationActionInstall, assetKey, key, func() operationResult {
		defer cancel()
		d.Logger.Info("Attempting to resolve installation method", "asset_type", req.AssetType, "asset_id", req.AssetID, "version", req.Version)
		switch req.AssetType {
		case types.AssetTypeMap:
			return operationResult{assetInstallResponse: d.installMapNow(opCtx, req.AssetID, req.Version, replaceOnConflict)}
		case types.AssetTypeMod:
			return operationResult{assetInstallResponse: d.installModNow(opCtx, req.AssetID, req.Version, req.Mod)}
		default:
			return operationResult{assetInstallResponse: d.installError(
				req.AssetType,
				req.AssetID,
				req.Version,
				types.ConfigData{},
				types.InstallErrorInvalidAssetType,
				"Invalid asset type",
				nil,
				"asset_type", req.AssetType, "asset_id", req.AssetID, "version", req.Version,
			)}
		}
	}, d.supersededOperationResult(operationActionInstall, req.AssetType, req.AssetID, req.Version), cancel)
	return result.assetInstallResponse
}

// installModNow handles the installation of a mod given its ID and version, including downloading, extracting, and updating the registry.
func (d *Downloader) installModNow(ctx context.Context, modId string, version string, modOptions *types.ModInstallOptions) types.AssetInstallResponse {
	d.Logger.Info("InstallMod started", "mod_id", modId, "version", version)
	if state, installed := d.getInstalledState(types.AssetTypeMod, modId); installed && state.version == version {
		return d.installWarn(
			types.AssetTypeMod,
			modId,
			version,
			types.ConfigData{},
			nil,
			fmt.Sprintf("%s already installed at requested version. No action taken.", assetTypeLabels[types.AssetTypeMod]),
			"asset_type", types.AssetTypeMod,
			"asset_id", modId,
			"version", version,
		)
	}
	skipDependencies := modOptions != nil && modOptions.SkipDependencies
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.installConfigError(types.AssetTypeMod, modId, version)
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
	if depErr := d.ensureModDependencies(ctx, modId, version, *versionInfo, skipDependencies); depErr != nil {
		return *depErr
	}

	d.Logger.Info("Downloading mod", "mod_id", modId, "version", version, "download_url", versionInfo.DownloadURL)
	// Pass in context to the download function so that it can be cancelled if the operation is no longer needed
	downloadResp := d.downloadTempZip(ctx, versionInfo.DownloadURL, modId)
	if downloadResp.Status == types.ResponseWarn {
		return d.installWarn(types.AssetTypeMod, modId, version, types.ConfigData{}, nil, downloadResp.Message, "mod_id", modId, "version", version)
	}
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorDownloadFailed, "Failed to download mod zip: "+downloadResp.Message, nil, "mod_id", modId, "version", version)
	}

	if err := d.verifySHA256(downloadResp.Path, versionInfo.SHA256); err != nil {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorChecksumFailed, "SHA-256 integrity check failed", err, "mod_id", modId, "version", version)
	}

	d.Logger.Info("Extracting mod", "mod_id", modId, "version", version, "temp_path", downloadResp.Path)
	// No context is passed here (for cancellation)
	extractResp := d.handleModExtract(downloadResp.Path, modId, version)
	if extractResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, extractResp.ErrorType, "Failed to extract mod zip: "+extractResp.Message, nil, "mod_id", modId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMod(modId, version, false)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after installing mod", "error", err)
	}
	d.Logger.Info("InstallMod completed", "mod_id", modId, "version", version)
	return d.installSuccess(types.AssetTypeMod, modId, version, types.ConfigData{}, "Mod installed successfully", "mod_id", modId, "version", version)
}

func (d *Downloader) ensureModDependencies(ctx context.Context, modID string, version string, versionInfo types.VersionInfo, skipDependencies bool) *types.AssetInstallResponse {
	if d.GetGameVersion != nil {
		gameVersionResp := d.GetGameVersion()
		if gameVersionResp.Status == types.ResponseSuccess {
			if subwayBuilderDep, ok := versionInfo.Dependencies["subway-builder"]; ok {
				constraint, err := semver.NewConstraint(strings.TrimPrefix(subwayBuilderDep, "v"))
				if err != nil {
					resp := d.installError(types.AssetTypeMod, modID, version, types.ConfigData{}, types.InstallErrorDependencyResolutionFailed, "Failed to parse mod dependency version constraint", err, "mod_id", modID, "dependency", "subway-builder", "constraint", subwayBuilderDep)
					return &resp
				}
				if !constraint.Check(semver.MustParse(strings.TrimPrefix(gameVersionResp.Version, "v"))) {
					resp := d.installError(types.AssetTypeMod, modID, version, types.ConfigData{}, types.InstallErrorDependencyResolutionFailed, "Mod is not compatible with current game version", nil, "mod_id", modID, "dependency", "subway-builder", "constraint", subwayBuilderDep, "game_version", gameVersionResp.Version)
					return &resp
				}
			}
		}
	}
	if skipDependencies {
		return nil
	}
	return d.installModDependencies(ctx, modID, version, versionInfo)
}

func (d *Downloader) installModDependencies(ctx context.Context, modID string, version string, versionInfo types.VersionInfo) *types.AssetInstallResponse {
	deps := d.ComputeDependencyList(modID, versionInfo)
	if deps.Status == types.ResponseError {
		resp := d.installError(types.AssetTypeMod, modID, version, types.ConfigData{}, types.InstallErrorDependencyResolutionFailed, "Failed to resolve mod dependencies", nil, "mod_id", modID, "version", version)
		return &resp
	}

	d.Logger.Info("Resolved mod dependencies", "mod_id", modID, "version", version, "dependencies", deps.InstallList)
	for depID, dep := range deps.InstallList {
		d.Logger.Info("Installing mod dependency", "mod_id", modID, "version", version, "dependency_id", depID, "dependency_version", dep.InstallCandidate.Version)
		if depID == modID {
			d.Logger.Info("Skipping installation of mod dependency that matches target mod", "mod_id", modID, "version", version)
			continue
		}
		if d.InstallDependency != nil {
			// Persist dependency subscription intent before installation so profile desired state is source-of-truth.
			depVersion := types.Version(dep.InstallCandidate.Version)
			d.InstallDependency(depID, types.AssetTypeMod, depVersion)
		}

		depInstallResp := d.installModNow(ctx, depID, dep.InstallCandidate.Version, &types.ModInstallOptions{SkipDependencies: true})
		if depInstallResp.Status == types.ResponseError {
			resp := d.installError(types.AssetTypeMod, modID, version, types.ConfigData{}, types.InstallErrorDependencyResolutionFailed, "Failed to install mod dependency", nil, "mod_id", modID, "dependency_id", depID, "dependency_version", dep.InstallCandidate.Version, "dependency_message", depInstallResp.Message)
			return &resp
		}
	}
	return nil
}

// installMapNow handles the installation of a map given its ID and version, including downloading, extracting, validating files, and updating the registry.
func (d *Downloader) installMapNow(ctx context.Context, mapId string, version string, replaceOnConflict bool) types.AssetInstallResponse {
	d.Logger.Info("InstallMap started", "map_id", mapId, "version", version)
	if state, installed := d.getInstalledState(types.AssetTypeMap, mapId); installed && state.version == version {
		return d.installWarn(
			types.AssetTypeMap,
			mapId,
			version,
			state.mapConfig,
			nil,
			fmt.Sprintf("%s already installed at requested version. No action taken.", assetTypeLabels[types.AssetTypeMap]),
			"asset_type", types.AssetTypeMap,
			"asset_id", mapId,
			"version", version,
		)
	}
	if !d.Config.GetConfig().Validation.IsValid() {
		return d.installConfigError(types.AssetTypeMap, mapId, version)
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
	// Pass in context to the download function so that it can be cancelled if the operation is no longer needed
	downloadResp := d.downloadTempZip(ctx, versionInfo.DownloadURL, mapId)
	if downloadResp.Status == types.ResponseWarn {
		return d.installWarn(types.AssetTypeMap, mapId, version, types.ConfigData{}, nil, downloadResp.Message, "map_id", mapId, "version", version)
	}
	if downloadResp.Status != types.ResponseSuccess {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorDownloadFailed, "Failed to download map zip: "+downloadResp.Message, nil, "map_id", mapId, "version", version)
	}

	if err := d.verifySHA256(downloadResp.Path, versionInfo.SHA256); err != nil {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, types.InstallErrorChecksumFailed, "SHA-256 integrity check failed", err, "map_id", mapId, "version", version)
	}

	cfg, configErrType, configErr := files.ValidateMapArchive(downloadResp.Path)
	if configErr != nil {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, configErrType, "Failed to inspect map archive", configErr, "map_id", mapId, "version", version)
	}

	_, conflictResp, handled := d.resolveMapInstallConflict(
		mapId,
		version,
		cfg,
		replaceOnConflict,
		true,
		"Map code conflict detected. Confirm replacement to continue.",
		"Failed to remove conflicting installed map before install",
		"map_id", mapId,
		"version", version,
	)
	if handled {
		os.Remove(downloadResp.Path)
		return conflictResp
	}

	d.Logger.Info("Extracting map", "map_id", mapId, "version", version, "temp_path", downloadResp.Path)
	// No context is passed here (for cancellation)
	extractResp := d.handleMapExtract(downloadResp.Path, mapId, version)
	if extractResp.Status == types.ResponseError {
		os.Remove(downloadResp.Path)
		return d.installError(types.AssetTypeMap, mapId, version, types.ConfigData{}, extractResp.ErrorType, "Failed to extract map zip: "+extractResp.Message, nil, "map_id", mapId, "version", version)
	}
	os.Remove(downloadResp.Path)
	d.Registry.AddInstalledMap(mapId, version, false, extractResp.Config)
	if err := d.Registry.WriteInstalledToDisk(); err != nil {
		d.Logger.Warn("Failed to persist installed state after installing map", "error", err)
	}
	d.Logger.Info("InstallMap completed", "map_id", mapId, "version", version)
	return extractResp
}

// downloadTempZip downloads a zip file from the given URL and saves it to a temporary location, returning the path or an error message.
func (d *Downloader) downloadTempZip(ctx context.Context, url string, itemId string) types.DownloadTempResponse {
	if err := os.MkdirAll(d.tempPath, os.ModePerm); err != nil {
		return d.throwDownloadError("Failed to create temp directory", err, "url", url)
	}

	file, err := os.CreateTemp(d.tempPath, "download-*.zip")
	if err != nil {
		return d.throwDownloadError("Failed to create temp file", err, "url", url)
	}
	tempPath := file.Name()
	keepTemp := false
	// Clean up temp zips on non-success paths (including cancellation) to avoid artifact buildup.
	defer func() {
		_ = file.Close()
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()
	zip, err := d.downloadRequest(ctx, url, d.Config.GetGithubToken())

	// Return a warning response instead of an error on cancellation
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return d.toDownloadResponse(d.warnResponse("Download cancelled by newer uninstall request", "url", url), "")
		}
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

	keepTemp = true
	return d.toDownloadResponse(d.successResponse("File downloaded successfully", "url", url), tempPath)
}

// downloadRequest performs an HTTP GET request to the specified URL, including an optional authentication for GitHub URL
func (d *Downloader) downloadRequest(ctx context.Context, downloadURL, githubToken string) (*http.Response, error) {
	return requests.GetWithGithubToken(downloaderHTTPClient, requests.GithubTokenRequestArgs{
		URL:                    downloadURL,
		GitHubToken:            githubToken,
		Context:                ctx,
		ShouldAuthenticateHost: isGitHubDownloadHost,
	})
}

var isGitHubDownloadHost = requests.IsGitHubHost

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
	reader, err := os.Open(paths.JoinLocalPath(cfgResult.Config.MetroMakerDataPath, "cities", "latest-cities.yml"))
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

// handleModExtract processes the downloaded mod zip file, extracts it to the appropriate location, and returns a success or error message.
func (d *Downloader) handleModExtract(filePath string, modId string, version string) types.AssetInstallResponse {
	return extractMod(d, filePath, modId, version)
}

// handleMapExtract processes map zip files and extracts the contract-specific files for local or downloaded assets.
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

func (d *Downloader) ComputeDependencyList(modId string, version types.VersionInfo) types.DependencyListResponse {
	deps := version.Dependencies
	installList := map[string]types.DependencyListEntry{modId: {Ranges: []string{}, InstallCandidate: version}}
	if len(deps) == 0 {
		return types.DependencyListResponse{
			GenericResponse: types.SuccessResponse("No dependencies to install."),
			InstallList:     installList,
		}
	}
	installList, err := d.recursivelyComputeDependencies(modId, version, installList, []string{})
	if err != nil {
		return types.DependencyListResponse{
			GenericResponse: types.ErrorResponse("Failed to compute dependencies: " + err.Error()),
			InstallList:     nil,
		}
	}
	return types.DependencyListResponse{
		GenericResponse: types.SuccessResponse("Successfully computed dependencies."),
		InstallList:     installList,
	}
}

func (d *Downloader) recursivelyComputeDependencies(modId string, version types.VersionInfo, installList map[string]types.DependencyListEntry, visited []string) (map[string]types.DependencyListEntry, error) {
	if slices.Contains(visited, modId) {
		cycle := append(slices.Clone(visited), modId)
		return nil, fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
	}

	visited = append(visited, modId)
	if existing, ok := installList[modId]; ok {
		existing.InstallCandidate = version
		installList[modId] = existing
	} else {
		installList[modId] = types.DependencyListEntry{
			Ranges:           []string{},
			InstallCandidate: version,
		}
	}

	if len(version.Dependencies) == 0 {
		return installList, nil
	}

	for depModID, depRange := range version.Dependencies {
		if slices.Contains(visited, depModID) {
			cycle := append(slices.Clone(visited), depModID)
			return nil, fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
		}

		depInfo, err := d.Registry.GetMod(depModID)
		if err != nil {
			return nil, fmt.Errorf("failed to get mod info for dependency %s: %w", depModID, err)
		}

		var versions []types.VersionInfo
		if depInfo.Update.Type == "github" {
			versions, err = d.Registry.GetVersions("github", depInfo.Update.Repo)
		} else {
			versions, err = d.Registry.GetVersions(depInfo.Update.Type, depInfo.Update.URL)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get versions for dependency %s: %w", depModID, err)
		}

		existingDep, exists := installList[depModID]
		if !exists {
			existingDep = types.DependencyListEntry{Ranges: []string{}}
		}

		normalizedRange := normalizeDependencyRange(depRange)
		if !slices.Contains(existingDep.Ranges, normalizedRange) {
			existingDep.Ranges = append(existingDep.Ranges, normalizedRange)
		}

		candidate, err := resolveDependencyCandidate(depModID, existingDep.Ranges, versions)
		if err != nil {
			return nil, err
		}

		candidateChanged := !exists || types.NormalizeSemver(existingDep.InstallCandidate.Version) != types.NormalizeSemver(candidate.Version)
		existingDep.InstallCandidate = candidate
		installList[depModID] = existingDep

		if candidateChanged {
			installList, err = d.recursivelyComputeDependencies(depModID, candidate, installList, visited)
			if err != nil {
				return nil, err
			}
		}
	}

	return installList, nil
}

func normalizeDependencyRange(versionRange string) string {
	trimmed := strings.TrimSpace(versionRange)
	if trimmed == "" {
		return "*"
	}
	return trimmed
}

func resolveDependencyCandidate(modID string, ranges []string, versions []types.VersionInfo) (types.VersionInfo, error) {
	if len(versions) == 0 {
		return types.VersionInfo{}, fmt.Errorf("no versions found for dependency %s", modID)
	}

	constraints := make([]*semver.Constraints, 0, len(ranges))
	for _, versionRange := range ranges {
		constraint, err := semver.NewConstraint(normalizeDependencyRange(versionRange))
		if err != nil {
			return types.VersionInfo{}, fmt.Errorf("invalid version range %q for dependency %s: %w", versionRange, modID, err)
		}
		constraints = append(constraints, constraint)
	}

	var candidate *types.VersionInfo
	var candidateSemver *semver.Version
	for i := range versions {
		parsedVersion, err := semver.NewVersion(strings.TrimPrefix(types.NormalizeSemver(versions[i].Version), "v"))
		if err != nil {
			continue
		}

		compatible := true
		for _, constraint := range constraints {
			if !constraint.Check(parsedVersion) {
				compatible = false
				break
			}
		}
		if !compatible {
			continue
		}

		if candidate == nil || parsedVersion.GreaterThan(candidateSemver) {
			candidate = &versions[i]
			candidateSemver = parsedVersion
		}
	}

	if candidate == nil {
		return types.VersionInfo{}, fmt.Errorf("conflicting version requirements for dependency %s: %v", modID, ranges)
	}

	return *candidate, nil
}
