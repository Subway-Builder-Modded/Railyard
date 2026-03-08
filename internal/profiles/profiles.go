package profiles

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"railyard/internal/downloader"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/types"
	"railyard/internal/utils"
)

type UserProfiles struct {
	state      types.UserProfilesState
	Logger     logger.Logger
	Registry   *registry.Registry
	Downloader *downloader.Downloader
	mu         sync.Mutex
	loaded     bool
}

const serviceName = "UserProfiles"

var (
	ErrProfileNotFound           = errors.New("profile not found")
	ErrInvalidSubscriptionAction = errors.New("invalid subscription action")
	ErrInvalidAssetType          = errors.New("invalid asset type")
	ErrInvalidVersion            = errors.New("invalid version")
	ErrProfilesNotLoaded         = errors.New("profiles state not loaded")
	ErrActiveProfileMissing      = errors.New("active profile missing from loaded state")
)

func NewUserProfiles(r *registry.Registry, d *downloader.Downloader, l logger.Logger) *UserProfiles {
	return &UserProfiles{
		Logger:     l,
		Registry:   r,
		Downloader: d,
	}
}

func (s *UserProfiles) setState(state types.UserProfilesState) {
	s.state = state
	s.loaded = true
}

func WriteUserProfilesState(state types.UserProfilesState) error {
	return files.WriteJSON(paths.UserProfilesPath(), "user profiles", state)
}

func ReadUserProfilesState() (types.UserProfilesState, error) {
	return files.ReadJSON[types.UserProfilesState](
		paths.UserProfilesPath(),
		"user profiles",
		files.JSONReadOptions{
			AllowMissing: true,
			AllowEmpty:   true,
		},
	)
}

// LoadProfiles loads profile state from disk and validates it, bootstrapping to defaults if missing or empty
func (s *UserProfiles) LoadProfiles() (activeProfile types.UserProfile, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("loadProfiles", "loaded", s.loaded)
	if s.loaded {
		return s.resolveActiveProfile()
	}

	state, err := ReadUserProfilesState()
	if err != nil {
		return types.UserProfile{}, err
	}

	// If no profiles exist on disk, initialize with default profile
	if len(state.Profiles) == 0 {
		s.Logger.Warn("No existing profiles found, bootstrapping with default profile")
		bootstrapped := types.InitialProfilesState()
		if err := WriteUserProfilesState(bootstrapped); err != nil {
			return types.UserProfile{}, err
		}
		s.setState(bootstrapped)
		return s.resolveActiveProfile()
	}

	validatedState, err := types.ValidateState(state)
	if err != nil {
		return types.UserProfile{}, err
	}

	s.setState(validatedState)
	return s.resolveActiveProfile()
}

func (s *UserProfiles) ResetUserProfiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("ResetUserProfiles", "num_profiles", len(s.state.Profiles))

	defaultState := types.InitialProfilesState()
	s.setState(defaultState)
	return WriteUserProfilesState(defaultState)
}

// QuarantineUserProfiles moves the user profiles file to a "quarantined" path in the same directory
// If the source file is missing, it is treated as a no-op.
func (s *UserProfiles) QuarantineUserProfiles() (success bool, backupPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("quarantineUserProfiles")

	return paths.QuarantineFile(paths.UserProfilesPath(), s.Logger)
}

// GetActiveProfile returns the active profile from loaded in-memory state.
// Callers must ensure LoadProfiles has completed successfully first.
func (s *UserProfiles) GetActiveProfile() (activeProfile types.UserProfile, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("GetActiveProfile")
	profile, resolveErr := s.resolveActiveProfile()
	if resolveErr != nil {
		s.Logger.Error("Failed to get active profile", resolveErr, "active_profile_id", s.state.ActiveProfileID)
		return types.UserProfile{}, resolveErr
	}
	return profile, nil
}

func (s *UserProfiles) resolveActiveProfile() (activeProfile types.UserProfile, err error) {
	if !s.loaded {
		return types.UserProfile{}, ErrProfilesNotLoaded
	}
	profile, ok := s.state.Profiles[s.state.ActiveProfileID]
	if !ok {
		return types.UserProfile{}, fmt.Errorf("%w: %q", ErrActiveProfileMissing, s.state.ActiveProfileID)
	}

	return profile, nil
}

// UpdateSubscriptions mutates the runtime state of the specified profile's subscriptions
func (s *UserProfiles) UpdateSubscriptions(req types.UpdateSubscriptionsRequest) (types.UpdateSubscriptionsResult, error) {
	s.logRequest("UpdateSubscriptions", "profile_id", req.ProfileID, "action", req.Action, "asset_count", len(req.Assets), "force_sync", req.ForceSync)

	s.mu.Lock()
	result, err := s.updateProfileSubscriptions(req)
	s.mu.Unlock()
	if err != nil {
		return types.UpdateSubscriptionsResult{}, err
	}

	if req.ForceSync {
		if err := s.SyncSubscriptions(req.ProfileID); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (s *UserProfiles) updateProfileSubscriptions(req types.UpdateSubscriptionsRequest) (types.UpdateSubscriptionsResult, error) {
	stateCopy := copyProfilesState(s.state)
	profile, ok := stateCopy.Profiles[req.ProfileID]
	if !ok {
		profileErr := fmt.Errorf("%w: %q", ErrProfileNotFound, req.ProfileID)
		s.Logger.Error("Profile not found", profileErr, "profile_id", req.ProfileID)
		return types.UpdateSubscriptionsResult{}, profileErr
	}

	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)

	operations := make([]types.SubscriptionOperation, 0, len(req.Assets))
	for assetID, item := range req.Assets {
		operation, opErr := applySubscriptionMutation(&profile, req.Action, strings.TrimSpace(assetID), item)
		if opErr != nil {
			s.Logger.Error("Failed to apply subscription mutation", opErr, "asset_id", assetID, "asset_type", item.Type, "action", req.Action)
			return types.UpdateSubscriptionsResult{}, opErr
		}
		if operation != nil {
			operations = append(operations, *operation)
		}
	}

	stateCopy.Profiles[req.ProfileID] = profile
	if req.ForceSync {
		if err := WriteUserProfilesState(stateCopy); err != nil {
			return types.UpdateSubscriptionsResult{}, err
		}
	}

	s.setState(stateCopy)
	result := types.UpdateSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  types.ResponseSuccess,
			Message: "subscriptions updated",
		},
		Profile:    profile,
		Persisted:  req.ForceSync,
		Operations: operations,
	}
	s.Logger.LogResponse(
		"Updated subscriptions",
		result.GenericResponse,
		"profile_id", req.ProfileID,
		"operation_count", len(operations),
		"persisted", req.ForceSync,
	)
	return result, nil
}

func syncActionError(action types.SubscriptionAction, assetType types.AssetType, assetID string, response types.GenericResponse) error {
	if response.Status == types.ResponseSuccess {
		return nil
	}
	return fmt.Errorf("%s %s %q failed with status=%s: %s", action, assetType, assetID, response.Status, response.Message)
}

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
) (*types.SubscriptionOperation, error) {
	switch item.Type {
	case types.AssetTypeMap:
		return mutateSubscriptionMap(profile.Subscriptions.Maps, action, assetID, item)
	case types.AssetTypeMod:
		return mutateSubscriptionMap(profile.Subscriptions.Mods, action, assetID, item)
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidAssetType, item.Type)
	}
}

func mutateSubscriptionMap(
	target map[string]string,
	action types.SubscriptionAction,
	assetID string,
	item types.SubscriptionUpdateItem,
) (*types.SubscriptionOperation, error) {
	switch action {
	case types.SubscriptionActionSubscribe:
		versionText := strings.TrimSpace(string(item.Version))
		if !types.IsValidSemverVersion(types.Version(versionText)) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidVersion, versionText)
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
		return nil, fmt.Errorf("%w: %q", ErrInvalidSubscriptionAction, action)
	}
}

// Helper struct to capture which functions are required to update subscriptions for a specific asset type, generic on the installed asset info type (T) and the manifest type (U)
type assetSyncArgs[T any, U any] struct {
	assetType     types.AssetType                                            // The type of asset being synced: map or mod (or others in the future).
	subscriptions map[string]string                                          // The desired subscription state for the profile, keyed by asset ID and valued by version.
	installedArgs installedVersionArgs[T]                                    // Non-shared installed-version resolver args.
	availableArgs availableVersionArgs[U]                                    // Non-shared available-version resolver args.
	install       func(assetID string, version string) types.GenericResponse // The function to call to install the asset (using the downloader).
	uninstall     func(assetID string) types.GenericResponse                 // The function to call to uninstall the asset (using the downloader).
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

// SyncSubscriptions iterates through a profile's subscriptions and attempts to reconcile the state of asset installtion on disk to the desired state in the profile by installing/uninstalling maps and mods as needed.
// Errors are collected and returned as a single error at the end, but the function will attempt reconciliation even if one or more of the individual install/uninstall operations fail or if the desired profile contains unavailable versions
func (s *UserProfiles) SyncSubscriptions(profileID string) error {
	s.logRequest("SyncSubscriptions", "profile_id", profileID)

	s.mu.Lock()
	profile, ok := s.state.Profiles[profileID]
	// Read a snapshot of current subscriptions at invocation time
	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)
	s.mu.Unlock()

	// This should not occur under calls from UpdateSubscriptions (or the startup call)
	if !ok {
		profileErr := fmt.Errorf("%w: %q", ErrProfileNotFound, profileID)
		s.Logger.Error("Profile not found for sync", profileErr, "profile_id", profileID)
		return profileErr
	}

	mapArgs := assetSyncArgs[types.InstalledMapInfo, types.MapManifest]{
		assetType:     types.AssetTypeMap,
		subscriptions: profile.Subscriptions.Maps,
		installedArgs: installedVersionArgs[types.InstalledMapInfo]{
			getInstalledAssetsFn: s.Registry.GetInstalledMaps,
			idFn:                 func(item types.InstalledMapInfo) string { return item.ID },
			versionFn:            func(item types.InstalledMapInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.MapManifest]{
			getManifestsFn: s.Registry.GetMaps,
			idFn:           func(item types.MapManifest) string { return item.ID },
			updateTypeFn:   func(item types.MapManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.MapManifest) string { return updateSource(item.Update) },
			getVersionsFn:  s.Registry.GetVersions,
		},
		install: func(assetID string, version string) types.GenericResponse {
			return s.Downloader.InstallMap(assetID, version).GenericResponse
		},
		uninstall: s.Downloader.UninstallMap,
	}

	modArgs := assetSyncArgs[types.InstalledModInfo, types.ModManifest]{
		assetType:     types.AssetTypeMod,
		subscriptions: profile.Subscriptions.Mods,
		installedArgs: installedVersionArgs[types.InstalledModInfo]{
			getInstalledAssetsFn: s.Registry.GetInstalledMods,
			idFn:                 func(item types.InstalledModInfo) string { return item.ID },
			versionFn:            func(item types.InstalledModInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.ModManifest]{
			getManifestsFn: s.Registry.GetMods,
			idFn:           func(item types.ModManifest) string { return item.ID },
			updateTypeFn:   func(item types.ModManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.ModManifest) string { return updateSource(item.Update) },
			getVersionsFn:  s.Registry.GetVersions,
		},
		install:   s.Downloader.InstallMod,
		uninstall: s.Downloader.UninstallMod,
	}

	syncErrors := make([]error, 0)
	// Run sync for each asset type in sequence
	syncErrors = append(syncErrors, syncAssetSubscriptions(mapArgs)...)
	syncErrors = append(syncErrors, syncAssetSubscriptions(modArgs)...)
	return errors.Join(syncErrors...)
}

func syncAssetSubscriptions[T any, U any](args assetSyncArgs[T, U]) []error {
	errs := make([]error, 0)
	installedVersion := buildVersionIndexFromItems(args.installedArgs)
	availableVersions := buildAvailableVersionIndex(args.availableArgs, args.subscriptions, args.assetType, &errs)

	for assetID, version := range args.subscriptions {
		versionText := strings.TrimSpace(version)
		// If the desired version is already installed, skip to the next asset
		if current, ok := installedVersion[assetID]; ok && current == versionText {
			continue
		}
		// Check if desired version is available according to the registry before attempting installation
		if !isVersionAvailable(availableVersions, assetID, versionText) {
			errs = append(errs, fmt.Errorf("%s %s %q failed: version %q is not available", types.SubscriptionActionSubscribe, args.assetType, assetID, versionText))
			continue
		}
		// If a different version is installed for this asset ID, uninstall it first
		if current, ok := installedVersion[assetID]; ok && current != versionText {
			uninstallResp := args.uninstall(assetID)
			if err := syncActionError(types.SubscriptionActionUnsubscribe, args.assetType, assetID, uninstallResp); err != nil {
				errs = append(errs, err)
				continue
			}
			delete(installedVersion, assetID)
		}
		response := args.install(assetID, versionText)
		// If installation fails, record the error but continue
		if err := syncActionError(types.SubscriptionActionSubscribe, args.assetType, assetID, response); err != nil {
			errs = append(errs, err)
			continue
		}
		installedVersion[assetID] = versionText
	}

	// Check for installed assets that are no longer subscribed and attempt uninstallation
	for assetID := range installedVersion {
		if _, ok := args.subscriptions[assetID]; ok {
			continue
		}
		response := args.uninstall(assetID)
		// If uninstallation fails, record the error but continue
		if err := syncActionError(types.SubscriptionActionUnsubscribe, args.assetType, assetID, response); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// buildVersionIndexFromItems makes use of the registry to build an index of installed assets
func buildVersionIndexFromItems[T any](args installedVersionArgs[T]) map[string]string {
	items := args.getInstalledAssetsFn()
	versions := make(map[string]string, len(items))
	for _, item := range items {
		versions[args.idFn(item)] = args.versionFn(item)
	}
	return versions
}

// buildAvailableVersionIndex makes use of the registry to build an index of available versions for each asset to which the profile is subscribed
func buildAvailableVersionIndex[U any](
	availableArgs availableVersionArgs[U],
	subscriptions map[string]string,
	assetType types.AssetType,
	syncErrors *[]error,
) map[string]map[string]struct{} {
	available := make(map[string]map[string]struct{})
	manifestByID := make(map[string]U)
	// Collect all available manifests and index by assetID for lookup
	for _, manifest := range availableArgs.getManifestsFn() {
		manifestByID[availableArgs.idFn(manifest)] = manifest
	}

	for assetID := range subscriptions {
		// If a particular assetID is not found in the registry's available manifests, skip and consider it to be "unavailable"
		manifest, ok := manifestByID[assetID]
		if !ok {
			continue
		}

		// Determine which versions are available for this asset, based on its update configuration
		versions, err := availableArgs.getVersionsFn(
			availableArgs.updateTypeFn(manifest),
			availableArgs.updateSourceFn(manifest),
		)
		if err != nil {
			*syncErrors = append(*syncErrors, fmt.Errorf("failed to resolve available versions for %s %q: %w", assetType, assetID, err))
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

func updateSource(update types.UpdateConfig) string {
	if update.Type == "github" {
		return update.Repo
	}
	return update.URL
}

// logRequest is a helper for consistent structured logging of service method calls and parameters
func (s *UserProfiles) logRequest(method string, attrs ...any) {
	base := []any{"service", serviceName}
	s.Logger.Info(fmt.Sprintf("Handling method: %s", method), append(base, attrs...)...)
}
