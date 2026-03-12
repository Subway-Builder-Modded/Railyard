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

// UpdateSubscriptions mutates the specified profile's subscriptions and persists the new state.
// ForceSync controls whether reconciliation/install-uninstall routines are run after persistence.
func (s *UserProfiles) UpdateSubscriptions(req types.UpdateSubscriptionsRequest) types.UpdateSubscriptionsResult {
	s.logRequest("UpdateSubscriptions", "profile_id", req.ProfileID, "action", req.Action, "asset_count", len(req.Assets), "force_sync", req.ForceSync)

	s.mu.Lock()
	result := s.updateProfileSubscriptions(req)
	s.mu.Unlock()
	if result.Status == types.ResponseError {
		return result
	}

	if req.ForceSync {
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
			result.Message = "Subscriptions updated with sync warnings"
			result.Errors = append(result.Errors, syncResult.Errors...)
		}
	}

	return result
}

// UpdateAllSubscriptionsToLatest resolves the latest available registry versions for all current profile subscriptions,
// updates those that are behind, persists updates to disk, and runs sync/install-uninstall routines.
func (s *UserProfiles) UpdateAllSubscriptionsToLatest(profileID string) types.UpdateSubscriptionsResult {
	s.logRequest("UpdateAllSubscriptionsToLatest", "profile_id", profileID)

	s.mu.Lock()
	profile := s.state.Profiles[profileID]
	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)
	s.mu.Unlock()

	requiredUpdates, resultWarnings := s.resolveLatestSubscriptionUpdates(profileID, profile)
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

	// If no updates are required, return early
	if len(requiredUpdates) == 0 {
		result := types.UpdateSubscriptionsResult{
			GenericResponse: types.GenericResponse{
				Status:  types.ResponseSuccess,
				Message: "All subscriptions already at latest version; no updates applied",
			},
			Profile:    profile,
			Persisted:  false,
			Operations: []types.SubscriptionOperation{},
			Errors:     resultWarnings,
		}
		if len(resultWarnings) > 0 {
			result.Status = types.ResponseWarn
			result.Message = fmt.Sprintf("no updates applied; skipped %d subscriptions during latest-version resolution", len(resultWarnings))
		}
		return result
	}

	// Otherwise, run the UpdateSubscriptions flow with the pre-calculated updates
	result := s.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: profileID,
		Assets:    requiredUpdates,
		Action:    types.SubscriptionActionSubscribe,
		ForceSync: true,
	})

	if len(resultWarnings) > 0 {
		if result.Status == types.ResponseSuccess {
			result.Status = types.ResponseWarn
		}
		result.Message = fmt.Sprintf("Updated %d subscriptions; skipped %d subscriptions during latest-version resolution", len(result.Operations), len(resultWarnings))
		result.Errors = append(result.Errors, resultWarnings...)
	}

	return result
}

// ===== Registry Helpers ===== //

func (s *UserProfiles) resolveLatestSubscriptionUpdates(profileID string, profile types.UserProfile) (map[string]types.SubscriptionUpdateItem, []types.UserProfilesError) {
	updates := make(map[string]types.SubscriptionUpdateItem)
	warnings := make([]types.UserProfilesError, 0)

	latestAssetUpdates(latestSubscriptionArgs[types.MapManifest]{
		assetType:     types.AssetTypeMap,
		subscriptions: profile.Subscriptions.Maps,
		getManifests:  s.Registry.GetMaps,
		idFn:          func(m types.MapManifest) string { return m.ID },
		updateFn:      func(m types.MapManifest) types.UpdateConfig { return m.Update },
	}, profileID, s.Registry.GetVersions, updates, &warnings)

	latestAssetUpdates(latestSubscriptionArgs[types.ModManifest]{
		assetType:     types.AssetTypeMod,
		subscriptions: profile.Subscriptions.Mods,
		getManifests:  s.Registry.GetMods,
		idFn:          func(m types.ModManifest) string { return m.ID },
		updateFn:      func(m types.ModManifest) types.UpdateConfig { return m.Update },
	}, profileID, s.Registry.GetVersions, updates, &warnings)

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
	profile, ok := stateCopy.Profiles[req.ProfileID]
	if !ok {
		profileErr := userProfilesError(req.ProfileID, "", "", types.ErrorProfileNotFound, fmt.Sprintf("Profile %q not found", req.ProfileID))
		s.Logger.Error("Profile not found", profileErr, "profile_id", req.ProfileID)
		return newUpdateSubscriptionsResult(
			types.ResponseError,
			"profile not found",
			types.UserProfile{},
			false,
			[]types.SubscriptionOperation{},
			[]types.UserProfilesError{profileErr},
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
	if err := WriteUserProfilesState(stateCopy); err != nil {
		return newUpdateSubscriptionsResult(
			types.ResponseError,
			"Failed to persist subscriptions",
			profile,
			false,
			operations,
			[]types.UserProfilesError{
				updateSubscriptionError(req.ProfileID, "", "", types.ErrorPersistFailed, fmt.Errorf("Failed to persist subscriptions: %w", err)),
			},
		)
	}

	s.setState(stateCopy)
	result := newUpdateSubscriptionsResult(
		types.ResponseSuccess,
		"Subscriptions updated",
		profile,
		true,
		operations,
		[]types.UserProfilesError{},
	)
	s.Logger.LogResponse(
		"Updated subscriptions",
		result.GenericResponse,
		"profile_id", req.ProfileID,
		"operation_count", len(operations),
		"persisted", result.Persisted,
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
