package profiles

import (
	"fmt"
	"strings"

	"railyard/internal/logger"
	"railyard/internal/types"
)

// SyncSubscriptions iterates through a profile's subscriptions and attempts to reconcile the state of asset installation on disk to the desired state in the profile by installing/uninstalling maps and mods as needed.
func (s *UserProfiles) SyncSubscriptions(profileID string, replaceOnConflict bool, skipDependencyInstall bool) types.SyncSubscriptionsResult {
	s.logRequest("SyncSubscriptions", "profile_id", profileID)

	profile, snapshotVersion, profileErr := s.profileSnapshot(profileID)
	if profileErr != nil {
		s.Logger.Error("Profile not found for sync", profileErr, "profile_id", profileID)
		result := syncResultBase(types.ResponseError, "Profile not found for sync", profileID)
		result.Errors = []types.UserProfilesError{*profileErr}
		return result
	}

	mapArgs := s.buildMapSyncArgs(profile, func() bool { return s.isSnapshotStale(snapshotVersion) }, replaceOnConflict)
	modArgs := s.buildModSyncArgs(profile, func() bool { return s.isSnapshotStale(snapshotVersion) }, skipDependencyInstall)

	syncErrors := make([]types.UserProfilesError, 0)
	operations := make([]types.SubscriptionOperation, 0)
	assetsToPurge := make([]assetPurgeArgs, 0)

	// Run sync for each asset type in sequence.
	s.Logger.Info("Syncing map subscriptions", "profile_id", profileID, "subscription_count", len(profile.Subscriptions.Maps))
	mapOperations, mapErrors, invalidMaps, mapStale := syncAssetSubscriptions(s.Logger, profileID, mapArgs)
	operations = append(operations, mapOperations...)
	syncErrors = append(syncErrors, mapErrors...)
	assetsToPurge = append(assetsToPurge, invalidMaps...)

	s.Logger.Info("Syncing mod subscriptions", "profile_id", profileID, "subscription_count", len(profile.Subscriptions.Mods))
	modOperations, modErrors, invalidMods, modStale := syncAssetSubscriptions(s.Logger, profileID, modArgs)
	operations = append(operations, modOperations...)
	syncErrors = append(syncErrors, modErrors...)
	assetsToPurge = append(assetsToPurge, invalidMods...)

	if mapStale || modStale {
		s.Logger.Warn("Subscription sync cancelled due to newer profile update", "profile_id", profileID)
		staleWarning := userProfilesError(
			profileID,
			"",
			"",
			types.ErrorSyncSuperseded,
			"",
			"Sync superseded by newer subscription update",
		)
		result := syncResultBase(types.ResponseWarn, "Sync cancelled by newer subscription update", profileID)
		result.Operations = operations
		result.Errors = []types.UserProfilesError{staleWarning}
		return result
	}

	purgeOperations, purgeErrors := s.applyPurgeOperations(profileID, assetsToPurge)
	if len(purgeOperations) > 0 {
		s.Logger.Warn("Purged invalid subscriptions after sync failures", "profile_id", profileID, "purge_count", len(purgeOperations))
		operations = append(operations, purgeOperations...)
	}
	syncErrors = append(syncErrors, purgeErrors...)

	if len(syncErrors) > 0 {
		s.Logger.Warn("Subscription sync completed with errors", "error_count", len(syncErrors))
		result := syncResultBase(types.ResponseError, fmt.Sprintf("subscription sync completed with %d error(s)", len(syncErrors)), profileID)
		result.Operations = operations
		result.Errors = syncErrors
		return result
	}

	if len(purgeOperations) > 0 {
		s.Logger.Warn("Subscription sync completed with purge warnings", "purge_count", len(purgeOperations))
		result := syncResultBase(types.ResponseWarn, fmt.Sprintf("subscription sync auto-purged %d invalid subscription(s)", len(purgeOperations)), profileID)
		result.Operations = operations
		return result
	}

	result := syncResultBase(types.ResponseSuccess, "subscriptions synced", profileID)
	result.Operations = operations
	return result
}

// assetPurgeArgs captures the information needed to attempt a purge of an invalid subscription
type assetPurgeArgs struct {
	assetType       types.AssetType
	assetID         string
	expectedVersion string
	errorCode       types.DownloaderErrorType
}

// Helper struct to capture which functions are required to update subscriptions for a specific asset type, generic on the installed asset info type (T) and the manifest type (U).
type assetSyncArgs[T any, U any] struct {
	assetType     types.AssetType                                                 // The type of asset being synced: map or mod (or others in the future).
	subscriptions map[string]string                                               // The desired subscription state for the profile, keyed by asset ID and valued by version.
	isStale       func() bool                                                     // Returns true when the sync snapshot is stale due to a newer profile update.
	installedArgs installedVersionArgs[T]                                         // Non-shared installed-version resolver args.
	availableArgs availableVersionArgs[U]                                         // Non-shared available-version resolver args.
	install       func(assetID string, version string) types.AssetInstallResponse // The function to call to install the asset (using the downloader).
	uninstall     func(assetID string) types.AssetUninstallResponse               // The function to call to uninstall the asset (using the downloader).
}

// Helper struct to capture what is needed to resolve installed versions for a specific asset type via the registry.
type installedVersionArgs[T any] struct {
	getInstalledAssetsFn func() []T
	idFn                 func(T) string
	versionFn            func(T) string
}

// Helper struct to capture what is needed to resolve available versions for a specific asset type via the registry.
type availableVersionArgs[U any] struct {
	getManifestsFn func() []U
	idFn           func(U) string
	updateTypeFn   func(U) string
	updateSourceFn func(U) string
	getVersionsFn  func(string, string) ([]types.VersionInfo, error)
}

// TODO: Consolidate this into a generic argument builder using types.AssetType to reduce duplication
func (s *UserProfiles) buildMapSyncArgs(profile types.UserProfile, isStale func() bool, replaceOnConflict bool) assetSyncArgs[types.InstalledMapInfo, types.MapManifest] {
	return assetSyncArgs[types.InstalledMapInfo, types.MapManifest]{
		assetType:     types.AssetTypeMap,
		subscriptions: profile.Subscriptions.Maps,
		isStale:       isStale,
		installedArgs: installedVersionArgs[types.InstalledMapInfo]{
			getInstalledAssetsFn: s.Registry.GetRemoteInstalledMaps,
			idFn:                 func(item types.InstalledMapInfo) string { return item.ID },
			versionFn:            func(item types.InstalledMapInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.MapManifest]{
			getManifestsFn: s.Registry.GetMaps,
			idFn:           func(item types.MapManifest) string { return item.ID },
			updateTypeFn:   func(item types.MapManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.MapManifest) string { return item.Update.Source() },
			getVersionsFn:  s.Registry.GetVersions,
		},
		install: func(assetID string, version string) types.AssetInstallResponse {
			return s.Downloader.InstallAsset(types.InstallAssetRequest{
				AssetType: types.AssetTypeMap,
				AssetID:   assetID,
				Version:   version,
				Map: &types.MapInstallOptions{
					ReplaceOnConflict: replaceOnConflict,
				},
			})
		},
		uninstall: func(assetID string) types.AssetUninstallResponse {
			return s.Downloader.UninstallAsset(types.AssetTypeMap, assetID)
		},
	}
}

func (s *UserProfiles) buildModSyncArgs(profile types.UserProfile, isStale func() bool, skipDependencyInstall bool) assetSyncArgs[types.InstalledModInfo, types.ModManifest] {
	return assetSyncArgs[types.InstalledModInfo, types.ModManifest]{
		assetType:     types.AssetTypeMod,
		subscriptions: profile.Subscriptions.Mods,
		isStale:       isStale,
		installedArgs: installedVersionArgs[types.InstalledModInfo]{
			getInstalledAssetsFn: s.Registry.GetInstalledMods,
			idFn:                 func(item types.InstalledModInfo) string { return item.ID },
			versionFn:            func(item types.InstalledModInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.ModManifest]{
			getManifestsFn: s.Registry.GetMods,
			idFn:           func(item types.ModManifest) string { return item.ID },
			updateTypeFn:   func(item types.ModManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.ModManifest) string { return item.Update.Source() },
			getVersionsFn:  s.Registry.GetVersions,
		},
		install: func(assetID string, version string) types.AssetInstallResponse {
			return s.Downloader.InstallAsset(types.InstallAssetRequest{
				AssetType: types.AssetTypeMod,
				AssetID:   assetID,
				Version:   version,
				Mod: &types.ModInstallOptions{
					SkipDependencies: skipDependencyInstall,
				},
			})
		},
		uninstall: func(assetID string) types.AssetUninstallResponse {
			return s.Downloader.UninstallAsset(types.AssetTypeMod, assetID)
		},
	}
}

// syncAssetSubscriptions is a generic type helper that performs the core logic of syncing subscriptions for a given asset type, with generic arguments corresponding to the asset's installed info type (T) and manifest type (U).
func syncAssetSubscriptions[T any, U any](log logger.Logger, profileID string, args assetSyncArgs[T, U]) ([]types.SubscriptionOperation, []types.UserProfilesError, []assetPurgeArgs, bool) {
	errs := make([]types.UserProfilesError, 0)
	operations := make([]types.SubscriptionOperation, 0)
	assetsToPurge := make([]assetPurgeArgs, 0)
	checkStale := func() bool {
		// If the snapshot is stale, we should stop processing immediately to avoid making unwanted changes based on an outdated profile state.
		return args.isStale != nil && args.isStale()
	}
	assetType := args.assetType
	installedVersion := buildVersionIndexFromItems(args.installedArgs)
	availableVersions := buildAvailableVersionIndex(args.availableArgs, profileID, args.subscriptions, assetType, &errs)

	log.Info("Built version indices for sync",
		"asset_type", args.assetType,
		"installed_count", len(installedVersion),
		"available_count", len(availableVersions),
	)

	for assetID, version := range args.subscriptions {
		if checkStale() {
			log.Warn("Stopping sync loop due to stale subscription snapshot", "asset_type", args.assetType)
			return operations, errs, assetsToPurge, true
		}
		versionText := strings.TrimSpace(version)
		// If the desired version is already installed, skip to the next asset.
		if current, ok := installedVersion[assetID]; ok && current == versionText {
			log.Info("Asset already at desired version, skipping", "asset_type", args.assetType, "asset_id", assetID, "version", versionText)
			continue
		}

		// Check if desired version is available according to the registry before attempting installation.
		if !isVersionAvailable(availableVersions, assetID, versionText) {
			availableForAsset := availableVersions[assetID]
			availableKeys := make([]string, 0, len(availableForAsset))
			for k := range availableForAsset {
				availableKeys = append(availableKeys, k)
			}
			log.Warn("Desired version not available",
				"asset_type", args.assetType,
				"asset_id", assetID,
				"desired_version", versionText,
				"available_versions", availableKeys,
			)
			errs = append(errs, userProfilesError(profileID, assetID, args.assetType, types.ErrorLookupFailed, "", fmt.Sprintf("Subscribe %s %q failed: version %q is not available", args.assetType, assetID, versionText)))
			continue
		}

		// If a different version is installed for this asset ID, uninstall it first.
		if current, ok := installedVersion[assetID]; ok && current != versionText {
			log.Info("Uninstalling previous version before update", "asset_type", args.assetType, "asset_id", assetID, "current_version", current, "target_version", versionText)
			uninstallResp := args.uninstall(assetID)
			if err := syncUninstallActionError(types.SubscriptionActionUnsubscribe, args.assetType, assetID, uninstallResp); err != nil {
				errs = append(errs, syncUninstallFailedError(profileID, assetID, args.assetType, uninstallResp, err))
				continue
			}
			operations = append(operations, types.SubscriptionOperation{
				AssetID: assetID,
				Type:    args.assetType,
				Action:  types.SubscriptionActionUnsubscribe,
				Version: types.Version(current),
			})
			delete(installedVersion, assetID)
		}

		log.Info("Installing asset", "asset_type", args.assetType, "asset_id", assetID, "version", versionText)
		response := args.install(assetID, versionText)
		if response.Status == types.ResponseWarn {
			// Occurs when installation skipped due to a newer subscription update (different version) or a cancellation (from a newer uninstall request).
			// These should be treated as warnings, not errors, since this is an expected set of events.
			log.Warn("Install skipped during sync", "asset_type", args.assetType, "asset_id", assetID, "version", versionText, "message", response.Message)
			continue
		}
		// If installation fails, record the error but continue.
		if err := syncInstallActionError(types.SubscriptionActionSubscribe, args.assetType, assetID, response); err != nil {
			log.Error("Install failed during sync", err, "asset_type", args.assetType, "asset_id", assetID, "version", versionText, "install_error_code", response.ErrorType)
			if types.AutoPurgeDownloadErrors(response.ErrorType) {
				log.Warn("Queuing invalid subscription for purge", "asset_type", args.assetType, "asset_id", assetID, "version", versionText, "install_error_code", response.ErrorType)
				assetsToPurge = append(assetsToPurge, assetPurgeArgs{
					assetType:       args.assetType,
					assetID:         assetID,
					expectedVersion: versionText,
					errorCode:       response.ErrorType,
				})
				continue
			}
			errs = append(errs, syncInstallFailedError(profileID, assetID, args.assetType, response, err))
			continue
		}
		log.Info("Successfully installed asset", "asset_type", args.assetType, "asset_id", assetID, "version", versionText)
		installedVersion[assetID] = versionText
		operations = append(operations, types.SubscriptionOperation{
			AssetID: assetID,
			Type:    args.assetType,
			Action:  types.SubscriptionActionSubscribe,
			Version: types.Version(versionText),
		})
	}

	// Check for installed assets that are no longer subscribed and attempt uninstallation.
	for assetID, currentVersion := range installedVersion {
		if checkStale() {
			log.Warn("Stopping uninstall loop due to stale subscription snapshot", "asset_type", args.assetType)
			return operations, errs, assetsToPurge, true
		}
		if _, ok := args.subscriptions[assetID]; ok {
			continue
		}
		log.Info("Uninstalling asset no longer subscribed", "asset_type", args.assetType, "asset_id", assetID)
		response := args.uninstall(assetID)
		// If uninstallation fails, record the error but continue.
		if err := syncUninstallActionError(types.SubscriptionActionUnsubscribe, args.assetType, assetID, response); err != nil {
			errs = append(errs, syncUninstallFailedError(profileID, assetID, args.assetType, response, err))
			continue
		}
		operations = append(operations, types.SubscriptionOperation{
			AssetID: assetID,
			Type:    args.assetType,
			Action:  types.SubscriptionActionUnsubscribe,
			Version: types.Version(currentVersion),
		})
	}

	return operations, errs, assetsToPurge, false
}

func (s *UserProfiles) applyPurgeOperations(profileID string, args []assetPurgeArgs) ([]types.SubscriptionOperation, []types.UserProfilesError) {
	if len(args) == 0 {
		return []types.SubscriptionOperation{}, []types.UserProfilesError{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.state.Profiles[profileID]
	if !ok {
		// This really should not happen unless the user is able to delete profiles while a sync is in-flight
		err := profileNotFoundError(profileID)
		return []types.SubscriptionOperation{}, []types.UserProfilesError{err}
	}

	purgeErrors := make([]types.UserProfilesError, 0)
	assets := make(map[string]types.SubscriptionUpdateItem, len(args))

	for _, arg := range args {
		currentVersion, exists := subscriptionVersion(profile, arg.assetType, arg.assetID)
		if !exists {
			continue
		}
		// This could happen if the subscription was modified after the sync started (e.g. via a new request to profiles)
		if currentVersion != arg.expectedVersion {
			s.Logger.Info(
				"Skipping purge due to stale subscription version",
				"profile_id", profileID,
				"asset_type", arg.assetType,
				"asset_id", arg.assetID,
				"expected_version", arg.expectedVersion,
				"current_version", currentVersion,
			)
			continue
		}

		assets[arg.assetID] = types.SubscriptionUpdateItem{Type: arg.assetType}
		s.Logger.Warn(
			"Purging invalid subscription",
			"profile_id", profileID,
			"asset_type", arg.assetType,
			"asset_id", arg.assetID,
			"version", arg.expectedVersion,
			"install_error_code", arg.errorCode,
		)
	}

	// No valid assets to purge after re-check
	if len(assets) == 0 {
		return []types.SubscriptionOperation{}, purgeErrors
	}

	result := s.updateProfileSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: profileID,
		Assets:    assets,
		Action:    types.SubscriptionActionUnsubscribe,
		ApplyMode: types.UpdateSubscriptionsPersistOnly,
	})
	if result.Status == types.ResponseError {
		return []types.SubscriptionOperation{}, append(purgeErrors, result.Errors...)
	}

	return result.Operations, purgeErrors
}

func subscriptionVersion(profile types.UserProfile, assetType types.AssetType, assetID string) (string, bool) {
	switch assetType {
	case types.AssetTypeMap:
		version, ok := profile.Subscriptions.Maps[assetID]
		return version, ok
	case types.AssetTypeMod:
		version, ok := profile.Subscriptions.Mods[assetID]
		return version, ok
	}

	panic(fmt.Sprintf("unsupported asset type %q", assetType))
}

// buildVersionIndexFromItems makes use of the registry to build an index of installed assets.
func buildVersionIndexFromItems[T any](args installedVersionArgs[T]) map[string]string {
	items := args.getInstalledAssetsFn()
	versions := make(map[string]string, len(items))
	for _, item := range items {
		versions[args.idFn(item)] = args.versionFn(item)
	}
	return versions
}

// buildAvailableVersionIndex makes use of the registry to build an index of available versions for each asset to which the profile is subscribed.
func buildAvailableVersionIndex[U any](
	availableArgs availableVersionArgs[U],
	profileID string,
	subscriptions map[string]string,
	assetType types.AssetType,
	syncErrors *[]types.UserProfilesError,
) map[string]map[string]struct{} {
	available := make(map[string]map[string]struct{})
	manifestByID := make(map[string]U)

	// Collect all available manifests and index by assetID for lookup.
	for _, manifest := range availableArgs.getManifestsFn() {
		manifestByID[availableArgs.idFn(manifest)] = manifest
	}

	for assetID := range subscriptions {
		// If a particular assetID is not found in the registry's available manifests, skip and consider it to be "unavailable".
		manifest, ok := manifestByID[assetID]
		if !ok {
			continue
		}

		// Determine which versions are available for this asset, based on its update configuration.
		versions, err := availableArgs.getVersionsFn(
			availableArgs.updateTypeFn(manifest),
			availableArgs.updateSourceFn(manifest),
		)
		if err != nil {
			*syncErrors = append(*syncErrors, updateSubscriptionError(profileID, assetID, assetType, types.ErrorLookupFailed, fmt.Errorf("Failed to resolve available versions for %s %q: %w", assetType, assetID, err)))
			continue
		}

		available[assetID] = make(map[string]struct{}, len(versions))
		for _, version := range versions {
			available[assetID][strings.TrimSpace(version.Version)] = struct{}{}
		}
	}

	return available
}

func isVersionAvailable(available map[string]map[string]struct{}, assetID string, version string) bool {
	versions, ok := available[assetID]
	if !ok {
		return false
	}
	_, ok = versions[strings.TrimSpace(version)]
	return ok
}
