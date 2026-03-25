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
)

type UserProfiles struct {
	state        types.UserProfilesState
	stateVersion uint64
	Logger       logger.Logger
	Registry     *registry.Registry
	Config       *config.Config
	Downloader   *downloader.Downloader
	mu           sync.Mutex
	loaded       bool
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
	// Counter for in-memory state versioning to detect stale snapshots
	s.stateVersion++
	s.loaded = true
}

func (s *UserProfiles) logRequest(method string, attrs ...any) {
	base := []any{"service", serviceName}
	s.Logger.Info(fmt.Sprintf("Handling method: %s", method), append(base, attrs...)...)
}

// ===== Request Results ===== //

func updateResultBase(
	requestType types.UpdateSubscriptionRequestType,
	status types.Status,
	message string,
) types.UpdateSubscriptionsResult {
	return types.UpdateSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  status,
			Message: message,
		},
		RequestType:    requestType,
		PendingUpdates: []types.PendingSubscriptionUpdate{},
		Operations:     []types.SubscriptionOperation{},
		Errors:         []types.UserProfilesError{},
		Conflicts:      []types.MapCodeConflict{},
	}
}

func syncResultBase(
	status types.Status,
	message string,
	profileID string,
) types.SyncSubscriptionsResult {
	return types.SyncSubscriptionsResult{
		GenericResponse: types.GenericResponse{
			Status:  status,
			Message: message,
		},
		ProfileID:  profileID,
		Operations: []types.SubscriptionOperation{},
		Errors:     []types.UserProfilesError{},
	}
}

func profileNotFoundUpdateResult(
	profileErr *types.UserProfilesError,
	requestType types.UpdateSubscriptionRequestType,
	message string,
) types.UpdateSubscriptionsResult {
	result := updateResultBase(requestType, types.ResponseError, message)
	result.Errors = []types.UserProfilesError{*profileErr}
	return result
}

// ===== Request Errors ===== //

func userProfilesError(
	profileID, assetID string,
	assetType types.AssetType,
	errorType types.UserProfilesErrorType,
	downloadErrorType types.DownloaderErrorType,
	message string,
) types.UserProfilesError {
	return types.UserProfilesError{
		ProfileID:           profileID,
		AssetID:             assetID,
		AssetType:           assetType,
		ErrorType:           errorType,
		DownloaderErrorType: downloadErrorType,
		Message:             strings.TrimSpace(message),
	}
}

func profileNotFoundError(profileID string) types.UserProfilesError {
	return userProfilesError(profileID, "", "", types.ErrorProfileNotFound, "", fmt.Sprintf("Profile %q not found", profileID))
}

func profileStateErrorResult(message string, profile types.UserProfile, errs ...types.UserProfilesError) types.UserProfileResult {
	return types.UserProfileResult{
		GenericResponse: types.ErrorResponse(message),
		Profile:         profile,
		Errors:          errs,
	}
}

func profileFromState(state types.UserProfilesState, profileID string) (types.UserProfile, *types.UserProfilesError) {
	profile, ok := state.Profiles[profileID]
	if !ok {
		err := profileNotFoundError(profileID)
		return types.UserProfile{}, &err
	}
	return profile, nil
}

func profileSnapshotFromState(state types.UserProfilesState, profileID string) (types.UserProfile, *types.UserProfilesError) {
	profile, profileErr := profileFromState(state, profileID)
	if profileErr != nil {
		return types.UserProfile{}, profileErr
	}
	cloneProfileSubscriptions(&profile)
	return profile, nil
}

func (s *UserProfiles) profileSnapshot(profileID string) (types.UserProfile, uint64, *types.UserProfilesError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read a snapshot of current subscriptions at invocation time.
	profile, profileErr := profileSnapshotFromState(s.state, profileID)
	if profileErr != nil {
		return types.UserProfile{}, 0, profileErr
	}
	return profile, s.stateVersion, nil
}

func (s *UserProfiles) isSnapshotStale(snapshotVersion uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateVersion != snapshotVersion
}

func updateSubscriptionError(profileID, assetID string, assetType types.AssetType, errorType types.UserProfilesErrorType, err error) types.UserProfilesError {
	return userProfilesError(profileID, assetID, assetType, errorType, "", fmt.Sprintf("Failed update action: %v", err))
}

func conflictWarningResult(
	requestType types.UpdateSubscriptionRequestType,
	message string,
	profile types.UserProfile,
	conflicts []types.MapCodeConflict,
) types.UpdateSubscriptionsResult {
	result := updateResultBase(requestType, types.ResponseWarn, message)
	result.Profile = profile
	result.Conflicts = conflicts
	return result
}

func syncFailedError(profileID, assetID string, assetType types.AssetType, err error) types.UserProfilesError {
	return userProfilesError(profileID, assetID, assetType, types.ErrorSyncFailed, "", fmt.Sprintf("Failed sync action: %v", err))
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
		response.ErrorType,
		fmt.Sprintf("Failed sync action: %v", err),
	)
}

func syncUninstallFailedError(profileID, assetID string, assetType types.AssetType, response types.AssetUninstallResponse, err error) types.UserProfilesError {
	return userProfilesError(
		profileID,
		assetID,
		assetType,
		types.ErrorSyncFailed,
		response.ErrorType,
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
