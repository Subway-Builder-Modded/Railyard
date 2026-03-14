package profiles

import (
	"fmt"
	"strings"
	"sync"

	"railyard/internal/config"
	"railyard/internal/downloader"
	"railyard/internal/logger"
	"railyard/internal/registry"
	"railyard/internal/types"
	"railyard/internal/utils"
)

type UserProfiles struct {
	state      types.UserProfilesState
	Logger     logger.Logger
	Registry   *registry.Registry
	Config     *config.Config
	Downloader *downloader.Downloader
	mu         sync.Mutex
	loaded     bool
}

const serviceName = "UserProfiles"

func NewUserProfiles(r *registry.Registry, d *downloader.Downloader, l logger.Logger, c *config.Config) *UserProfiles {
	return &UserProfiles{
		Logger:     l,
		Registry:   r,
		Downloader: d,
		Config:     c,
	}
}

func (s *UserProfiles) setState(state types.UserProfilesState) {
	s.state = state
	s.loaded = true
}

func (s *UserProfiles) logRequest(method string, attrs ...any) {
	base := []any{"service", serviceName}
	s.Logger.Info(fmt.Sprintf("Handling method: %s", method), append(base, attrs...)...)
}

// ===== Request Results ===== //

func newUpdateSubscriptionsResult(
	status types.Status,
	message string,
	profile types.UserProfile,
	persisted bool,
	operations []types.SubscriptionOperation,
	profileErrors []types.UserProfilesError,
) types.UpdateSubscriptionsResult {
	return types.UpdateSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  status,
			Message: message,
		},
		Profile:    profile,
		Persisted:  persisted,
		Operations: operations,
		Errors:     profileErrors,
	}
}

func newSyncSubscriptionsResult(
	status types.Status,
	message string,
	profileID string,
	operations []types.SubscriptionOperation,
	profileErrors []types.UserProfilesError,
) types.SyncSubscriptionsResult {
	return types.SyncSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  status,
			Message: message,
		},
		ProfileID:  profileID,
		Operations: operations,
		Errors:     profileErrors,
	}
}

// ===== Request Errors ===== //

func userProfilesError(profileID, assetID string, assetType types.AssetType, errorType types.UserProfilesErrorType, message string) types.UserProfilesError {
	return types.UserProfilesError{
		ProfileID: profileID,
		AssetID:   assetID,
		AssetType: assetType,
		ErrorType: errorType,
		Message:   strings.TrimSpace(message),
	}
}

func profileNotFoundError(profileID string) types.UserProfilesError {
	return userProfilesError(profileID, "", "", types.ErrorProfileNotFound, fmt.Sprintf("Profile %q not found", profileID))
}

func profileFromState(state types.UserProfilesState, profileID string) (types.UserProfile, *types.UserProfilesError) {
	profile, ok := state.Profiles[profileID]
	if !ok {
		err := profileNotFoundError(profileID)
		return types.UserProfile{}, &err
	}
	return profile, nil
}

func (s *UserProfiles) snapshotProfile(profileID string) (types.UserProfile, *types.UserProfilesError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, profileErr := profileFromState(s.state, profileID)
	if profileErr != nil {
		return types.UserProfile{}, profileErr
	}

	profile.Subscriptions.Maps = utils.CloneMap(profile.Subscriptions.Maps)
	profile.Subscriptions.Mods = utils.CloneMap(profile.Subscriptions.Mods)
	return profile, nil
}

func updateSubscriptionError(profileID, assetID string, assetType types.AssetType, errorType types.UserProfilesErrorType, err error) types.UserProfilesError {
	return userProfilesError(profileID, assetID, assetType, errorType, fmt.Sprintf("Failed update action: %v", err))
}

func syncFailedError(profileID, assetID string, assetType types.AssetType, err error) types.UserProfilesError {
	return userProfilesError(profileID, assetID, assetType, types.ErrorSyncFailed, fmt.Sprintf("Failed sync action: %v", err))
}

func syncInstallFailedError(profileID, assetID string, assetType types.AssetType, response types.AssetInstallResponse, err error) types.UserProfilesError {
	if response.ErrorType == "" { // Programmer error; we should always have some sort of ErrorCode
		panic(fmt.Sprintf("syncInstallFailedError received empty install error code for %s %q", assetType, assetID))
	}
	return userProfilesError(
		profileID,
		assetID,
		assetType,
		types.UserProfilesErrorType(response.ErrorType),
		fmt.Sprintf("Failed sync action: %v", err),
	)
}

func syncUninstallActionError(action types.SubscriptionAction, assetType types.AssetType, assetID string, response types.AssetUninstallResponse) error {
	if response.Status != types.ResponseError {
		return nil
	}
	return fmt.Errorf("%s %s %q failed with status=%s code=%s: %s", action, assetType, assetID, response.Status, response.ErrorType, response.Message)
}

func syncInstallActionError(action types.SubscriptionAction, assetType types.AssetType, assetID string, response types.AssetInstallResponse) error {
	if response.Status != types.ResponseError {
		return nil
	}
	return fmt.Errorf("%s %s %q failed with status=%s code=%s: %s", action, assetType, assetID, response.Status, response.ErrorType, response.Message)
}

func (s *UserProfiles) archiveError(logMessage, responseMessage string, err error, attrs ...any) (types.GenericResponse, bool) {
	s.Logger.Error(logMessage, err, attrs...)
	return types.ErrorResponse(fmt.Errorf("%s: %w", responseMessage, err).Error()), false
}
