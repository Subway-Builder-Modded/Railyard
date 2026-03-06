package profiles

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/types"
	"railyard/internal/utils"
)

type UserProfiles struct {
	state  types.UserProfilesState
	Logger logger.Logger
	mu     sync.Mutex
	loaded bool
}

const serviceName = "UserProfiles"

var (
	ErrProfileNotFound           = errors.New("profile not found")
	ErrInvalidSubscriptionAction = errors.New("invalid subscription action")
	ErrInvalidAssetType          = errors.New("invalid asset type")
	ErrProfilesNotLoaded         = errors.New("profiles state not loaded")
	ErrActiveProfileMissing      = errors.New("active profile missing from loaded state")
)

func NewUserProfiles(l logger.Logger) *UserProfiles {
	return &UserProfiles{
		Logger: l,
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
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logRequest("UpdateSubscriptions", "profile_id", req.ProfileID, "action", req.Action, "asset_count", len(req.Assets), "force_sync", req.ForceSync)
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

// logRequest is a helper for consistent structured logging of service method calls and parameters
func (s *UserProfiles) logRequest(method string, attrs ...any) {
	base := []any{"service", serviceName}
	s.Logger.Info(fmt.Sprintf("Handling method: %s", method), append(base, attrs...)...)
}
