package profiles

import (
	"errors"
	"fmt"
	"strings"

	"railyard/internal/types"
	"railyard/internal/utils"

	"golang.org/x/mod/semver"
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
					userProfilesErrorWithDownloadError(
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
					userProfilesErrorWithDownloadError(
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
		syncResult := s.SyncSubscriptions(req.ProfileID)
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

// UpdateAllSubscriptionsToLatest resolves the latest available registry versions for all current profile subscriptions,
// updates those that are behind, persists updates to disk, and runs sync/install-uninstall routines.
func (s *UserProfiles) UpdateAllSubscriptionsToLatest(req types.UpdateAllSubscriptionsToLatestRequest) types.UpdateSubscriptionsResult {
	s.logRequest("UpdateAllSubscriptionsToLatest", "profile_id", req.ProfileID, "apply", req.Apply)

	profile, requiredUpdates, resultWarnings, profileErr := s.resolveLatestUpdatesForProfile(req.ProfileID)
	if profileErr != nil {
		return types.UpdateSubscriptionsResult{
			GenericResponse: types.GenericResponse{
				Status:  types.ResponseError,
				Message: "Profile not found",
			},
			RequestType:  types.LatestCheck,
			HasUpdates:   false,
			PendingCount: 0,
			Applied:      false,
			Profile:      types.UserProfile{},
			Persisted:    false,
			Operations:   []types.SubscriptionOperation{},
			Errors:       []types.UserProfilesError{*profileErr},
		}
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

	pendingCount := len(requiredUpdates)
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

		requestType := types.LatestCheck
		if req.Apply {
			requestType = types.LatestApply
		}
		return types.UpdateSubscriptionsResult{
			GenericResponse: types.GenericResponse{
				Status:  status,
				Message: message,
			},
			RequestType:  requestType,
			HasUpdates:   hasUpdates,
			PendingCount: pendingCount,
			Applied:      false,
			Profile:      profile,
			Persisted:    false,
			Operations:   []types.SubscriptionOperation{},
			Errors:       resultWarnings,
		}
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

	return types.UpdateSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  status,
			Message: message,
		},
		RequestType:  types.LatestApply,
		HasUpdates:   hasUpdates,
		PendingCount: pendingCount,
		Applied:      true,
		Profile:      updateResult.Profile,
		Persisted:    updateResult.Persisted,
		Operations:   updateResult.Operations,
		Errors:       errors,
	}
}

func (s *UserProfiles) resolveLatestUpdatesForProfile(profileID string) (types.UserProfile, map[string]types.SubscriptionUpdateItem, []types.UserProfilesError, *types.UserProfilesError) {
	profile, _, profileErr := s.profileSnapshot(profileID)
	if profileErr != nil {
		return types.UserProfile{}, map[string]types.SubscriptionUpdateItem{}, []types.UserProfilesError{}, profileErr
	}

	requiredUpdates, warnings := s.resolveLatestSubscriptionUpdates(profileID, profile)
	return profile, requiredUpdates, warnings, nil
}

// ===== Registry Helpers ===== //

func (s *UserProfiles) resolveLatestSubscriptionUpdates(profileID string, profile types.UserProfile) (map[string]types.SubscriptionUpdateItem, []types.UserProfilesError) {
	updates := make(map[string]types.SubscriptionUpdateItem)
	warnings := make([]types.UserProfilesError, 0)

	latestAssetUpdates(
		latestSubscriptionArgs[types.MapManifest]{
			assetType:     types.AssetTypeMap,
			subscriptions: profile.Subscriptions.Maps,
			getManifests:  s.Registry.GetMaps,
			idFn:          func(m types.MapManifest) string { return m.ID },
			updateFn:      func(m types.MapManifest) types.UpdateConfig { return m.Update },
		},
		profileID, s.Registry.GetVersions, updates, &warnings,
	)

	latestAssetUpdates(
		latestSubscriptionArgs[types.ModManifest]{
			assetType:     types.AssetTypeMod,
			subscriptions: profile.Subscriptions.Mods,
			getManifests:  s.Registry.GetMods,
			idFn:          func(m types.ModManifest) string { return m.ID },
			updateFn:      func(m types.ModManifest) types.UpdateConfig { return m.Update },
		},
		profileID, s.Registry.GetVersions, updates, &warnings,
	)

	return updates, warnings
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
	getVersionsFn func(string, string) ([]types.VersionInfo, error),
	updates map[string]types.SubscriptionUpdateItem,
	errors *[]types.UserProfilesError,
) {
	manifestUpdateByID := make(map[string]types.UpdateConfig)
	for _, manifest := range args.getManifests() {
		manifestUpdateByID[args.idFn(manifest)] = args.updateFn(manifest)
	}

	for assetID, currentVersion := range args.subscriptions {
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
		}
	}
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
	current := normalize(best)
	for _, version := range versions[1:] {
		other := normalize(version.Version)
		if semver.Compare(other, current) > 0 {
			current = other
			best = version.Version
		}
	}
	return best, nil
}

// ===== Runtime Mutation Helpers ===== //

func (s *UserProfiles) updateProfileSubscriptions(req types.UpdateSubscriptionsRequest) types.UpdateSubscriptionsResult {
	stateCopy := copyProfilesState(s.state)
	profile, profileErr := profileFromState(stateCopy, req.ProfileID)
	if profileErr != nil {
		s.Logger.Error("Profile not found", profileErr, "profile_id", req.ProfileID)
		return newUpdateSubscriptionsResult(
			types.ResponseError,
			"profile not found",
			false,
			types.UserProfile{},
			false,
			[]types.SubscriptionOperation{},
			[]types.UserProfilesError{*profileErr},
		)
	}

	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)

	operations := make([]types.SubscriptionOperation, 0, len(req.Assets))
	for assetID, item := range req.Assets {
		operation, opErr := applySubscriptionMutation(&profile, req.Action, strings.TrimSpace(assetID), item)
		if opErr != nil {
			s.Logger.Error("Failed to apply subscription mutation", *opErr, "asset_id", assetID, "asset_type", item.Type, "action", req.Action)
			return newUpdateSubscriptionsResult(
				types.ResponseError,
				"Failed to apply subscription mutation",
				false,
				profile,
				false,
				[]types.SubscriptionOperation{},
				[]types.UserProfilesError{*opErr},
			)
		}
		if operation != nil {
			operations = append(operations, *operation)
		}
	}

	stateCopy.Profiles[req.ProfileID] = profile
	if req.ForceSync {
		if err := WriteUserProfilesState(stateCopy); err != nil {
			return newUpdateSubscriptionsResult(
				types.ResponseError,
				"Failed to persist subscriptions",
				false,
				profile,
				false,
				operations,
				[]types.UserProfilesError{
					updateSubscriptionError(req.ProfileID, "", "", types.ErrorPersistFailed, fmt.Errorf("Failed to persist subscriptions: %w", err)),
				},
			)
		}
	}

	s.setState(stateCopy)
	result := newUpdateSubscriptionsResult(
		types.ResponseSuccess,
		"Subscriptions updated",
		true,
		profile,
		req.ForceSync,
		operations,
		[]types.UserProfilesError{},
	)
	s.Logger.LogResponse(
		"Updated subscriptions",
		result.GenericResponse,
		"profile_id", req.ProfileID,
		"operation_count", len(operations),
		"persisted", req.ForceSync,
	)
	return result
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

func applySubscriptionMutation(
	profile *types.UserProfile,
	action types.SubscriptionAction,
	assetID string,
	item types.SubscriptionUpdateItem,
) (*types.SubscriptionOperation, *types.UserProfilesError) {
	switch item.Type {
	case types.AssetTypeMap:
		return mutateSubscriptionMap(profile.Subscriptions.Maps, action, assetID, item)
	case types.AssetTypeMod:
		return mutateSubscriptionMap(profile.Subscriptions.Mods, action, assetID, item)
	default:
		err := userProfilesError("", assetID, item.Type, types.ErrorInvalidAssetType, fmt.Sprintf("Invalid asset type: %q", item.Type))
		return nil, &err
	}
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
			err := userProfilesError("", assetID, item.Type, types.ErrorInvalidVersion, fmt.Sprintf("Invalid version: %q", versionText))
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
		err := userProfilesError("", assetID, item.Type, types.ErrorInvalidAction, fmt.Sprintf("Invalid subscription action: %q", action))
		return nil, &err
	}
}
