package profiles

import (
	"errors"
	"fmt"
	"strings"

	"railyard/internal/types"
	"railyard/internal/utils"

	semver "github.com/Masterminds/semver/v3"
)

// ===== Profile Mutations ===== //

// UpdateSubscriptions mutates the runtime state of the specified profile's subscriptions
func (s *UserProfiles) UpdateSubscriptions(req types.UpdateSubscriptionsRequest) types.UpdateSubscriptionsResult {
	s.logRequest("UpdateSubscriptions", "profile_id", req.ProfileID, "action", req.Action, "asset_count", len(req.Assets), "force_sync", req.ForceSync)

	s.mu.Lock()
	result := s.updateProfileSubscriptions(req)
	s.mu.Unlock()
	if result.Status == types.ResponseError {
		return result
	}

	if req.ForceSync {
		cancelErrors := make([]types.UserProfilesError, 0)
		cancelFailed := false
		for _, operation := range result.Operations {
			if operation.Action != types.SubscriptionActionUnsubscribe {
				continue
			}
			cancelResp := s.Downloader.UninstallAsset(operation.Type, operation.AssetID)
			if cancelResp.Status == types.ResponseError {
				s.Logger.Warn("Failed to enqueue uninstall cancellation", "asset_type", operation.Type, "asset_id", operation.AssetID, "message", cancelResp.Message)
				cancelErrors = append(
					cancelErrors,
					userProfilesError(
						req.ProfileID,
						operation.AssetID,
						operation.Type,
						types.ErrorSyncFailed,
						cancelResp.ErrorType,
						cancelResp.Message,
					),
				)
				cancelFailed = true
				continue
			}
			if cancelResp.Status == types.ResponseWarn {
				cancelErrors = append(
					cancelErrors,
					userProfilesError(
						req.ProfileID,
						operation.AssetID,
						operation.Type,
						types.ErrorSyncFailed,
						cancelResp.ErrorType,
						cancelResp.Message,
					),
				)
			}
		}
		if len(cancelErrors) > 0 {
			result.Errors = append(result.Errors, cancelErrors...)
			if cancelFailed {
				result.Status = types.ResponseError
				result.Message = "Failed to cancel pending installs"
			} else if result.Status == types.ResponseSuccess {
				result.Status = types.ResponseWarn
				result.Message = "Subscriptions updated with cancellation warnings"
			}
		}

		// Unsubscribe requests already issue direct uninstall/cancel operations above. Skip the full sync routine to avoid redundant processing
		if req.Action == types.SubscriptionActionUnsubscribe {
			return result
		}

		// TODO: Implement per-profile request coalescing so burst frontend updates reconcile once
		// against the latest desired subscriptions state instead of running multiple stale snapshots.
		syncResult := s.SyncSubscriptions(req.ProfileID, req.ReplaceOnConflict, req.SkipDependencyInstall)
		if syncResult.Status == types.ResponseError {
			result.Status = types.ResponseError
			result.Message = "Failed to sync subscriptions"
			result.Errors = append(result.Errors, syncResult.Errors...)
			return result
		}
		if syncResult.Status == types.ResponseWarn {
			result.Status = types.ResponseWarn
			if strings.TrimSpace(syncResult.Message) != "" {
				result.Message = syncResult.Message
			} else {
				result.Message = "Subscriptions updated with sync warnings"
			}
			result.Errors = append(result.Errors, syncResult.Errors...)
		}
	}

	return result
}

// UpdateSubscriptionsToLatest resolves the latest available registry versions for current profile subscriptions,
// updates those that are behind, persists updates to disk, and runs sync/install-uninstall routines.
func (s *UserProfiles) UpdateSubscriptionsToLatest(req types.UpdateSubscriptionsToLatestRequest) types.UpdateSubscriptionsResult {
	s.logRequest(
		"UpdateSubscriptionsToLatest",
		"profile_id", req.ProfileID,
		"apply", req.Apply,
		"target_count", len(req.Targets),
	)

	requestType := types.LatestCheck
	if req.Apply {
		requestType = types.LatestApply
	}

	profile, requiredUpdates, pendingUpdates, resultWarnings, profileErr := s.resolveLatestUpdatesForProfile(req.ProfileID, req.Targets)
	if profileErr != nil {
		return profileNotFoundUpdateResult(profileErr, requestType, "Profile not found")
	}

	for _, warn := range resultWarnings {
		s.Logger.Warn(
			"Skipped subscription while resolving latest version",
			"profile_id", warn.ProfileID,
			"asset_id", warn.AssetID,
			"asset_type", warn.AssetType,
			"error_type", warn.ErrorType,
			"error", warn.Message,
		)
	}

	pendingCount := len(pendingUpdates)
	hasUpdates := pendingCount > 0

	if !req.Apply || !hasUpdates {
		status := types.ResponseSuccess
		message := "Resolved subscription update availability"
		if req.Apply && !hasUpdates {
			message = "All subscriptions already at latest version; no updates applied"
		}
		if len(resultWarnings) > 0 {
			status = types.ResponseWarn
			if req.Apply && !hasUpdates {
				message = fmt.Sprintf("no updates applied; skipped %d subscriptions during latest-version resolution", len(resultWarnings))
			} else {
				message = fmt.Sprintf("Resolved update availability with %d warning(s)", len(resultWarnings))
			}
		}

		result := updateResultBase(requestType, status, message)
		result.HasUpdates = hasUpdates
		result.PendingCount = pendingCount
		result.PendingUpdates = pendingUpdates
		result.Profile = profile
		result.Errors = resultWarnings
		return result
	}

	updateResult := s.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: req.ProfileID,
		Assets:    requiredUpdates,
		Action:    types.SubscriptionActionSubscribe,
		ForceSync: true,
	})

	status := updateResult.Status
	message := updateResult.Message
	errors := updateResult.Errors
	if len(resultWarnings) > 0 {
		if status == types.ResponseSuccess {
			status = types.ResponseWarn
		}
		message = fmt.Sprintf("Updated %d subscriptions; skipped %d subscriptions during latest-version resolution", len(updateResult.Operations), len(resultWarnings))
		errors = append(errors, resultWarnings...)
	}

	result := updateResultBase(types.LatestApply, status, message)
	result.HasUpdates = hasUpdates
	result.PendingCount = pendingCount
	result.PendingUpdates = pendingUpdates
	result.Applied = true
	result.Profile = updateResult.Profile
	result.Persisted = updateResult.Persisted
	result.Operations = updateResult.Operations
	result.Errors = errors
	return result
}

func (s *UserProfiles) resolveLatestUpdatesForProfile(
	profileID string,
	targets []types.SubscriptionUpdateTarget,
) (
	types.UserProfile,
	map[string]types.SubscriptionUpdateItem,
	[]types.PendingSubscriptionUpdate,
	[]types.UserProfilesError,
	*types.UserProfilesError,
) {
	profile, _, profileErr := s.profileSnapshot(profileID)
	if profileErr != nil {
		return types.UserProfile{}, map[string]types.SubscriptionUpdateItem{}, []types.PendingSubscriptionUpdate{}, []types.UserProfilesError{}, profileErr
	}

	requiredUpdates, pendingUpdates, warnings := s.resolveLatestSubscriptionUpdates(profileID, profile, targets)
	return profile, requiredUpdates, pendingUpdates, warnings, nil
}

// ===== Registry Helpers ===== //

func (s *UserProfiles) resolveLatestSubscriptionUpdates(
	profileID string,
	profile types.UserProfile,
	targets []types.SubscriptionUpdateTarget,
) (
	map[string]types.SubscriptionUpdateItem,
	[]types.PendingSubscriptionUpdate,
	[]types.UserProfilesError,
) {
	updates := make(map[string]types.SubscriptionUpdateItem)
	pendingUpdates := make([]types.PendingSubscriptionUpdate, 0)
	warnings := make([]types.UserProfilesError, 0)
	targetSet := makeTargetSet(targets)

	latestAssetUpdates(
		latestSubscriptionArgs[types.MapManifest]{
			assetType:     types.AssetTypeMap,
			subscriptions: profile.Subscriptions.Maps,
			getManifests:  s.Registry.GetMaps,
			idFn:          func(m types.MapManifest) string { return m.ID },
			updateFn:      func(m types.MapManifest) types.UpdateConfig { return m.Update },
		},
		profileID, targetSet, s.Registry.GetVersions, updates, &pendingUpdates, &warnings,
	)

	latestAssetUpdates(
		latestSubscriptionArgs[types.ModManifest]{
			assetType:     types.AssetTypeMod,
			subscriptions: profile.Subscriptions.Mods,
			getManifests:  s.Registry.GetMods,
			idFn:          func(m types.ModManifest) string { return m.ID },
			updateFn:      func(m types.ModManifest) types.UpdateConfig { return m.Update },
		},
		profileID, targetSet, s.Registry.GetVersions, updates, &pendingUpdates, &warnings,
	)

	return updates, pendingUpdates, warnings
}

type latestSubscriptionArgs[T any] struct {
	assetType     types.AssetType
	subscriptions map[string]string
	getManifests  func() []T
	idFn          func(T) string
	updateFn      func(T) types.UpdateConfig
}

func latestAssetUpdates[T any](
	args latestSubscriptionArgs[T],
	profileID string,
	targetSet map[assetVersionKey]struct{},
	getVersionsFn func(string, string) ([]types.VersionInfo, error),
	updates map[string]types.SubscriptionUpdateItem,
	pendingUpdates *[]types.PendingSubscriptionUpdate,
	errors *[]types.UserProfilesError,
) {
	manifestUpdateByID := make(map[string]types.UpdateConfig)
	for _, manifest := range args.getManifests() {
		manifestUpdateByID[args.idFn(manifest)] = args.updateFn(manifest)
	}

	for assetID, currentVersion := range args.subscriptions {
		// Check if the asset is within the requested update targets (if any were given)
		if !shouldUpdate(targetSet, args.assetType, assetID) {
			continue
		}

		update, ok := manifestUpdateByID[assetID]
		if !ok {
			*errors = append(*errors, updateSubscriptionError(
				profileID, assetID, args.assetType, types.ErrorLookupFailed,
				fmt.Errorf("Asset %q missing from registry manifests for %s", assetID, args.assetType),
			))
			continue
		}

		latestVersion, resolveErr := resolveLatestVersionForManifest(update, getVersionsFn)
		if resolveErr != nil {
			*errors = append(*errors, updateSubscriptionError(
				profileID, assetID, args.assetType, types.ErrorLookupFailed,
				fmt.Errorf("Failed to resolve latest version for %s %q: %w", args.assetType, assetID, resolveErr),
			))
			continue
		}

		if strings.TrimSpace(currentVersion) != latestVersion {
			updates[assetID] = types.SubscriptionUpdateItem{
				Type:    args.assetType,
				Version: types.Version(latestVersion),
			}
			*pendingUpdates = append(*pendingUpdates, types.PendingSubscriptionUpdate{
				AssetID:        assetID,
				Type:           args.assetType,
				CurrentVersion: types.Version(strings.TrimSpace(currentVersion)),
				LatestVersion:  types.Version(latestVersion),
			})
		}
	}
}

type assetVersionKey struct {
	AssetType types.AssetType
	AssetID   string
}

func makeTargetSet(targets []types.SubscriptionUpdateTarget) map[assetVersionKey]struct{} {
	targetSet := make(map[assetVersionKey]struct{}, len(targets))
	for _, target := range targets {
		targetSet[assetVersionKey{
			AssetType: target.Type,
			AssetID:   target.AssetID,
		}] = struct{}{}
	}
	return targetSet
}

func shouldUpdate(targetSet map[assetVersionKey]struct{}, assetType types.AssetType, assetID string) bool {
	if len(targetSet) == 0 {
		return true
	}
	_, ok := targetSet[assetVersionKey{
		AssetType: assetType,
		AssetID:   assetID,
	}]
	return ok
}

func resolveLatestVersionForManifest(
	update types.UpdateConfig,
	getVersionsFn func(string, string) ([]types.VersionInfo, error),
) (string, error) {
	versions, err := getVersionsFn(update.Type, update.Source())
	if err != nil {
		return "", fmt.Errorf("Failed to resolve versions: %w", err)
	}
	if len(versions) == 0 {
		return "", errors.New("No versions found")
	}

	// Assume Registry only contains valid semver versions and normalize with potential "v" prefix.
	normalize := func(v string) string {
		if strings.HasPrefix(v, "v") {
			return v
		}
		return "v" + v
	}

	best := versions[0].Version
	current, err := semver.NewVersion(strings.TrimPrefix(normalize(best), "v"))
	if err != nil {
		return "", fmt.Errorf("failed to parse initial semver version %q: %w", best, err)
	}
	for _, version := range versions[1:] {
		other, parseErr := semver.NewVersion(strings.TrimPrefix(normalize(version.Version), "v"))
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse semver version %q: %w", version.Version, parseErr)
		}
		if other.GreaterThan(current) {
			current = other
			best = version.Version
		}
	}
	return best, nil
}

// ===== Runtime Mutation Helpers ===== //

func (s *UserProfiles) updateProfileSubscriptions(req types.UpdateSubscriptionsRequest) types.UpdateSubscriptionsResult {
	stateCopy, profile, profileErr := s.resolveProfileFromCopy(req.ProfileID)
	if profileErr != nil {
		s.Logger.Error("Profile not found", profileErr, "profile_id", req.ProfileID)
		return profileNotFoundUpdateResult(profileErr, types.UpdateSubscriptions, "profile not found")
	}

	cloneProfileSubscriptions(&profile)

	operations := make([]types.SubscriptionOperation, 0, len(req.Assets))
	conflicts := make([]types.MapCodeConflict, 0)
	if req.Action == types.SubscriptionActionSubscribe && req.ForceSync {
		// Check for map code conflicts before applying any mutations to surface surface confirmation request to FE
		mapCodeConflicts := s.checkMapCodeConflicts(req)
		if len(mapCodeConflicts) > 0 && !req.ReplaceOnConflict {
			return conflictWarningResult(
				types.UpdateSubscriptions,
				"Map code conflict detected. Confirm replacement to continue.",
				profile,
				mapCodeConflicts,
			)
		}
		conflicts = mapCodeConflicts

		conflictOps, conflictErr := s.applyConflictReplacement(req.ProfileID, &profile, conflicts)
		if conflictErr != nil {
			message := "Failed to apply map conflict replacement"
			if conflictErr.DownloaderErrorType == types.InstallErrorMapCodeConflict {
				message = conflictErr.Message
			}
			result := updateResultBase(types.UpdateSubscriptions, types.ResponseError, message)
			result.Profile = profile
			result.Errors = []types.UserProfilesError{*conflictErr}
			return result
		}
		operations = appendOperations(operations, conflictOps)
	}

	for assetID, item := range req.Assets {
		operation, opErr := applySubscriptionMutation(&profile, req.Action, strings.TrimSpace(assetID), item)
		if opErr != nil {
			s.Logger.Error("Failed to apply subscription mutation", *opErr, "asset_id", assetID, "asset_type", item.Type, "action", req.Action)
			result := updateResultBase(types.UpdateSubscriptions, types.ResponseError, "Failed to apply subscription mutation")
			result.Profile = profile
			result.Errors = []types.UserProfilesError{*opErr}
			return result
		}
		operations = appendOperation(operations, operation)
	}

	if err := s.commitProfileMutation(&stateCopy, req.ProfileID, profile, req.ForceSync); err != nil {
		result := updateResultBase(types.UpdateSubscriptions, types.ResponseError, "Failed to persist subscriptions")
		result.Profile = profile
		result.Operations = operations
		result.Errors = []types.UserProfilesError{
			updateSubscriptionError(req.ProfileID, "", "", types.ErrorPersistFailed, fmt.Errorf("Failed to persist subscriptions: %w", err)),
		}
		return result
	}

	result := updateResultBase(types.UpdateSubscriptions, types.ResponseSuccess, "Subscriptions updated")
	result.Applied = true
	result.Profile = profile
	result.Persisted = req.ForceSync
	result.Operations = operations
	result.Conflicts = conflicts
	s.Logger.LogResponse(
		"Updated subscriptions",
		result.GenericResponse,
		"profile_id", req.ProfileID,
		"operation_count", len(operations),
		"persisted", req.ForceSync,
	)
	return result
}

// ImportAsset imports a local archive into installed state and wires it into the active profile as a local subscription.
func (s *UserProfiles) ImportAsset(req types.ImportAssetRequest) types.UpdateSubscriptionsResult {
	s.logRequest(
		"ImportAsset",
		"profile_id", req.ProfileID,
		"asset_type", req.AssetType,
		"replace_on_conflict", req.ReplaceOnConflict,
	)
	// Initial snapshot to validate profile existence before performing any operations.
	// Unlike remote subscription updates, we need to validate that the asset can be imported before mutating the profile state since we do not want to persist any subscription changes if the import fails.
	profile, _, profileErr := s.profileSnapshot(req.ProfileID)
	if profileErr != nil {
		return profileNotFoundUpdateResult(profileErr, types.ImportAsset, "Profile not found")
	}

	// Validate that the asset can be imported successfully
	importResp := s.Downloader.ImportAsset(req.AssetType, req.ZipPath, req.ReplaceOnConflict)
	if importResp.Status == types.ResponseError {
		err := userProfilesError(
			req.ProfileID,
			importResp.AssetID,
			req.AssetType,
			types.ErrorSyncFailed,
			importResp.ErrorType,
			importResp.Message,
		)
		result := updateResultBase(types.ImportAsset, types.ResponseError, importResp.Message)
		result.Profile = profile
		result.Errors = []types.UserProfilesError{err}
		return result
	}

	// If the import succeeded but recorded a warning about map code conflict, this must be raised to the frontend to confirm replaceent before profile mutation
	if importResp.Status == types.ResponseWarn && importResp.MapCodeConflict != nil && !req.ReplaceOnConflict {
		result := conflictWarningResult(
			types.ImportAsset,
			importResp.Message,
			profile,
			[]types.MapCodeConflict{*importResp.MapCodeConflict},
		)
		return result
	}

	// Re-lock and re-snapshot to perform profile mutation after import (given that import itself may take an extended period of time)
	s.mu.Lock()
	defer s.mu.Unlock()

	stateCopy, nextProfile, nextProfileErr := s.resolveProfileFromCopy(req.ProfileID)
	if nextProfileErr != nil {
		return profileNotFoundUpdateResult(nextProfileErr, types.ImportAsset, "Profile not found")
	}

	cloneProfileSubscriptions(&nextProfile)

	// There will be at most two subscription mutations to apply for an imported map asset: one to resolve map code conflicts if they exist, and one to subscribe to the newly imported map.
	// Apply conflict resolution first to ensure the profile is in a valid state to subscribe to the new asse\
	operations := make([]types.SubscriptionOperation, 0, 2)
	appliedConflicts := make([]types.MapCodeConflict, 0)
	if importResp.MapCodeConflict != nil && req.ReplaceOnConflict {
		conflictOperations, conflictErr := s.applyConflictReplacement(req.ProfileID, &nextProfile, []types.MapCodeConflict{*importResp.MapCodeConflict})
		if conflictErr != nil {
			message := "Failed to replace conflicting subscription"
			if conflictErr.DownloaderErrorType == types.InstallErrorMapCodeConflict {
				message = conflictErr.Message
			}
			result := updateResultBase(types.ImportAsset, types.ResponseError, message)
			result.Profile = nextProfile
			result.Errors = []types.UserProfilesError{*conflictErr}
			return result
		}
		operations = appendOperations(operations, conflictOperations)
		appliedConflicts = append(appliedConflicts, *importResp.MapCodeConflict)
	}

	// Then apply the subscription mutation to add the imported asset to the profile.
	// Each asset type will have two different sets of subscription collections, one for locally imported assets and one for remotely subscribed assets.
	localOperation, localErr := applySubscriptionMutation(
		&nextProfile,
		types.SubscriptionActionSubscribe,
		importResp.AssetID,
		types.SubscriptionUpdateItem{
			Type:    types.AssetTypeMap,
			Version: types.Version(importResp.Version),
			IsLocal: true,
		},
	)
	if localErr != nil {
		result := updateResultBase(types.ImportAsset, types.ResponseError, "Failed to add imported map to profile subscriptions")
		result.Profile = nextProfile
		result.Errors = []types.UserProfilesError{*localErr}
		return result
	}
	operations = appendOperation(operations, localOperation)

	// If all operations (optional conflict replacement + import subscription) were applied successfully, commit the profile mutation to persist the new subscriptions state with the imported asset.
	if err := s.commitProfileMutation(&stateCopy, req.ProfileID, nextProfile, true); err != nil {
		persistErr := updateSubscriptionError(
			req.ProfileID,
			importResp.AssetID,
			types.AssetTypeMap,
			types.ErrorPersistFailed,
			fmt.Errorf("failed to persist imported asset subscriptions: %w", err),
		)
		result := updateResultBase(types.ImportAsset, types.ResponseError, "Failed to persist imported asset subscriptions")
		result.Profile = nextProfile
		result.Operations = operations
		result.Errors = []types.UserProfilesError{persistErr}
		return result
	}

	status := types.ResponseSuccess
	message := "Asset imported and subscribed successfully"
	if importResp.Status == types.ResponseWarn {
		status = types.ResponseWarn
		message = importResp.Message
	}

	result := updateResultBase(types.ImportAsset, status, message)
	result.Applied = true
	result.Profile = nextProfile
	result.Persisted = true
	result.Operations = operations
	result.Conflicts = appliedConflicts
	return result
}

// checkMapCodeConflicts checks for potential map code conflicts between the maps in the update request and existing subscriptions.
func (s *UserProfiles) checkMapCodeConflicts(req types.UpdateSubscriptionsRequest) []types.MapCodeConflict {
	conflicts := make([]types.MapCodeConflict, 0)

	for assetID, item := range req.Assets {
		if item.Type != types.AssetTypeMap || item.IsLocal {
			continue
		}
		manifest, err := s.Registry.GetMap(assetID)
		if err != nil {
			continue
		}

		conflict, hasConflict := s.Downloader.FindMapCodeConflict(assetID, manifest.CityCode, true)
		if !hasConflict {
			continue
		}
		conflicts = append(conflicts, *conflict)
	}

	return conflicts
}

// copyProfilesState is a helper to create a deep copy of the profiles state prior to mutation
func copyProfilesState(source types.UserProfilesState) types.UserProfilesState {
	copied := types.UserProfilesState{
		ActiveProfileID: source.ActiveProfileID,
		Profiles:        make(map[string]types.UserProfile, len(source.Profiles)),
	}
	for id, profile := range source.Profiles {
		copied.Profiles[id] = profile
	}
	return copied
}

// resolveProfileFromCopy returns a mutable state copy and resolved profile for mutation paths.
func (s *UserProfiles) resolveProfileFromCopy(profileID string) (types.UserProfilesState, types.UserProfile, *types.UserProfilesError) {
	stateCopy := copyProfilesState(s.state)
	profile, profileErr := profileFromState(stateCopy, profileID)
	if profileErr != nil {
		return types.UserProfilesState{}, types.UserProfile{}, profileErr
	}
	return stateCopy, profile, nil
}

func (s *UserProfiles) commitProfileMutation(state *types.UserProfilesState, profileID string, profile types.UserProfile, persist bool) error {
	state.Profiles[profileID] = profile
	if persist {
		if err := WriteUserProfilesState(*state); err != nil {
			return err
		}
	}
	s.setState(*state)
	return nil
}

func cloneProfileSubscriptions(profile *types.UserProfile) {
	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)
	profile.Subscriptions.LocalMaps = utils.CloneMap(profile.Subscriptions.LocalMaps)
}

func appendOperation(operations []types.SubscriptionOperation, operation *types.SubscriptionOperation) []types.SubscriptionOperation {
	if operation == nil {
		return operations
	}
	return append(operations, *operation)
}

func appendOperations(operations []types.SubscriptionOperation, additional []types.SubscriptionOperation) []types.SubscriptionOperation {
	if len(additional) == 0 {
		return operations
	}
	return append(operations, additional...)
}

// applyConflictReplacement applies the necessary subscription mutations to resolve map code conflicts.
func (s *UserProfiles) applyConflictReplacement(
	profileID string,
	profile *types.UserProfile,
	conflicts []types.MapCodeConflict,
) ([]types.SubscriptionOperation, *types.UserProfilesError) {
	operations := make([]types.SubscriptionOperation, 0, len(conflicts))
	for _, conflict := range conflicts {
		// Vanilla map conflicts are not resolvable since we should not overwrite vanilla maps.
		// If this occurs it is either a registry error or an issue with an imported map's configuration
		if strings.HasPrefix(conflict.ExistingAssetID, "vanilla:") {
			err := userProfilesError(
				profileID,
				conflict.ExistingAssetID,
				conflict.ExistingAssetType,
				types.ErrorInvalidAction,
				types.InstallErrorMapCodeConflict,
				"Cannot replace a vanilla map city code",
			)
			return nil, &err
		}
		removedOperation, removeErr := removeMapConflictSubscription(profile, conflict)
		if removeErr != nil {
			return nil, removeErr
		}
		operations = appendOperation(operations, removedOperation)
	}
	return operations, nil
}

func applySubscriptionMutation(
	profile *types.UserProfile,
	action types.SubscriptionAction,
	assetID string,
	item types.SubscriptionUpdateItem,
) (*types.SubscriptionOperation, *types.UserProfilesError) {
	switch item.Type {
	// TODO: Generalize non-local/local split across all asset types instead of applying a special case for maps only.
	case types.AssetTypeMap:
		target := profile.Subscriptions.Maps
		if item.IsLocal {
			target = profile.Subscriptions.LocalMaps
		} else if action == types.SubscriptionActionUnsubscribe {
			if _, exists := profile.Subscriptions.LocalMaps[assetID]; exists {
				target = profile.Subscriptions.LocalMaps
			}
		}
		return mutateSubscriptionMap(target, action, assetID, item)
	case types.AssetTypeMod:
		return mutateSubscriptionMap(profile.Subscriptions.Mods, action, assetID, item)
	default:
		err := userProfilesError("", assetID, item.Type, types.ErrorInvalidAssetType, "", fmt.Sprintf("Invalid asset type: %q", item.Type))
		return nil, &err
	}
}

// removeMapConflictSubscription removes the subscription that is in conflict with a new map subscription to resolve map code conflicts.
func removeMapConflictSubscription(
	profile *types.UserProfile,
	conflict types.MapCodeConflict,
) (*types.SubscriptionOperation, *types.UserProfilesError) {
	target := profile.Subscriptions.Maps
	item := types.SubscriptionUpdateItem{
		Type: types.AssetTypeMap,
	}
	if conflict.ExistingIsLocal {
		target = profile.Subscriptions.LocalMaps
		item.IsLocal = true
	}
	return mutateSubscriptionMap(target, types.SubscriptionActionUnsubscribe, conflict.ExistingAssetID, item)
}

func mutateSubscriptionMap(
	target map[string]string,
	action types.SubscriptionAction,
	assetID string,
	item types.SubscriptionUpdateItem,
) (*types.SubscriptionOperation, *types.UserProfilesError) {
	switch action {
	case types.SubscriptionActionSubscribe:
		versionText := strings.TrimSpace(string(item.Version))
		if !types.IsValidSemverVersion(types.Version(versionText)) {
			err := userProfilesError("", assetID, item.Type, types.ErrorInvalidVersion, "", fmt.Sprintf("Invalid version: %q", versionText))
			return nil, &err
		}
		target[assetID] = versionText
		return &types.SubscriptionOperation{
			AssetID: assetID,
			Type:    item.Type,
			Action:  action,
			Version: types.Version(versionText),
		}, nil
	case types.SubscriptionActionUnsubscribe:
		removedVersion, exists := target[assetID]
		if !exists {
			return nil, nil
		}
		delete(target, assetID)
		return &types.SubscriptionOperation{
			AssetID: assetID,
			Type:    item.Type,
			Action:  action,
			Version: types.Version(strings.TrimSpace(removedVersion)),
		}, nil
	default:
		err := userProfilesError("", assetID, item.Type, types.ErrorInvalidAction, "", fmt.Sprintf("Invalid subscription action: %q", action))
		return nil, &err
	}
}
