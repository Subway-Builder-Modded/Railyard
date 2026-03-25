package profiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/types"
	"railyard/internal/utils"
	"strconv"
	"strings"
)

// ===== Profile I/O ===== //

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
func (s *UserProfiles) LoadProfiles() types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("loadProfiles", "loaded", s.loaded)
	if s.loaded {
		return s.resolveActiveProfile()
	}

	state, err := ReadUserProfilesState()
	if err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("failed to load profiles state"),
			Errors: []types.UserProfilesError{
				userProfilesError("", "", "", types.ErrorUnknown, "", "Failed to load profiles state: "+err.Error()),
			},
		}
	}

	// If no profiles exist on disk, initialize with default profile
	if len(state.Profiles) == 0 {
		s.Logger.Warn("No existing profiles found, bootstrapping with default profile")
		bootstrapped := types.InitialProfilesState()
		if err := WriteUserProfilesState(bootstrapped); err != nil {
			return types.UserProfileResult{
				GenericResponse: types.ErrorResponse("Failed to bootstrap default profiles"),
				Profile:         types.DefaultProfile(),
				Errors: []types.UserProfilesError{
					userProfilesError(types.DefaultProfileID, "", "", types.ErrorPersistFailed, "", "Failed to bootstrap default profiles: "+err.Error()),
				},
			}
		}
		s.setState(bootstrapped)
		return s.resolveActiveProfile()
	}

	validatedState, err := types.ValidateState(state)
	if err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Profiles state validation failed"),
			Errors: []types.UserProfilesError{
				userProfilesError("", "", "", types.ErrorUnknown, "", "Profiles state validation failed: "+err.Error()),
			},
		}
	}

	s.setState(validatedState)
	return s.resolveActiveProfile()
}

// QuarantineUserProfiles moves the user profiles file to a "quarantined" path in the same directory.
// If the source file is missing, it is treated as a no-op.
func (s *UserProfiles) QuarantineUserProfiles() (success bool, backupPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("quarantineUserProfiles")

	return paths.QuarantineFile(paths.UserProfilesPath(), s.Logger)
}

// ===== Profile State Resolution ===== //

// GetActiveProfile returns the active profile from loaded in-memory state.
// Callers must ensure LoadProfiles has completed successfully first.
func (s *UserProfiles) GetActiveProfile() types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("GetActiveProfile")
	result := s.resolveActiveProfile()
	if result.Status == types.ResponseError {
		s.Logger.MultipleError("Failed to get active profile", logger.AsErrors(result.Errors), "active_profile_id", s.state.ActiveProfileID)
	}
	return result
}

func (s *UserProfiles) resolveActiveProfile() types.UserProfileResult {
	if !s.loaded {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Profiles state not loaded"),
			Errors: []types.UserProfilesError{
				userProfilesError("", "", "", types.ErrorProfilesNotLoaded, "", "Profiles state not loaded"),
			},
		}
	}
	profile, ok := s.state.Profiles[s.state.ActiveProfileID]
	if !ok {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Active profile missing from loaded state"),
			Errors: []types.UserProfilesError{
				userProfilesError(s.state.ActiveProfileID, "", "", types.ErrorProfileNotFound, "", `Active profile missing from loaded state: "`+s.state.ActiveProfileID+`"`),
			},
		}
	}

	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("Active profile resolved"),
		Profile:         profile,
		Errors:          []types.UserProfilesError{},
	}
}

// ===== Profile State Mutation ===== //

// ResetUserProfiles deletes the existing profiles state and replaces it with the default. It is intended to be both a recovery mechanism for a corrupted state and as an option for users to reset their profiles.
func (s *UserProfiles) ResetUserProfiles() types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("ResetUserProfiles", "num_profiles", len(s.state.Profiles))

	defaultState := types.InitialProfilesState()
	s.setState(defaultState)
	active := defaultState.Profiles[defaultState.ActiveProfileID]
	if err := WriteUserProfilesState(defaultState); err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Failed to persist reset profiles state"),
			Profile:         active,
			Errors: []types.UserProfilesError{
				userProfilesError(active.ID, "", "", types.ErrorPersistFailed, "", "Failed to persist reset profiles state: "+err.Error()),
			},
		}
	}
	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("Profiles reset to defaults"),
		Profile:         active,
		Errors:          []types.UserProfilesError{},
	}
}

func (s *UserProfiles) UpdateSystemPreferences(prefs types.SystemPreferences) types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("UpdateSystemPreferences", "refresh_registry_on_startup", prefs.RefreshRegistryOnStartup, "extra_heap_size", prefs.ExtraHeapSize, "use_dev_tools", prefs.UseDevTools)

	nextState := types.UserProfilesState{
		ActiveProfileID: s.state.ActiveProfileID,
		Profiles:        make(map[string]types.UserProfile, len(s.state.Profiles)),
	}
	for id, p := range s.state.Profiles {
		nextState.Profiles[id] = p
	}

	profile := nextState.Profiles[s.state.ActiveProfileID]
	profile.SystemPreferences = prefs
	nextState.Profiles[s.state.ActiveProfileID] = profile

	validatedState, err := types.ValidateState(nextState)
	if err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Invalid system preferences"),
			Profile:         s.state.Profiles[s.state.ActiveProfileID],
			Errors: []types.UserProfilesError{
				userProfilesError(s.state.ActiveProfileID, "", "", types.ErrorUnknown, "", "Invalid system preferences: "+err.Error()),
			},
		}
	}

	s.setState(validatedState)

	if err := WriteUserProfilesState(validatedState); err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Failed to persist system preferences"),
			Profile:         validatedState.Profiles[s.state.ActiveProfileID],
			Errors: []types.UserProfilesError{
				userProfilesError(s.state.ActiveProfileID, "", "", types.ErrorPersistFailed, "", "Failed to persist system preferences: "+err.Error()),
			},
		}
	}

	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("System preferences updated"),
		Profile:         validatedState.Profiles[s.state.ActiveProfileID],
		Errors:          []types.UserProfilesError{},
	}
}

// UpdateUIPreferences updates the active profile UI preferences and persists the profile state.
func (s *UserProfiles) UpdateUIPreferences(uiPrefs types.UIPreferences) types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("UpdateUIPreferences", "theme", uiPrefs.Theme, "default_per_page", uiPrefs.DefaultPerPage)

	// Create a deep copy of the current state
	nextState := types.UserProfilesState{
		ActiveProfileID: s.state.ActiveProfileID,
		Profiles:        make(map[string]types.UserProfile, len(s.state.Profiles)),
	}
	for id, p := range s.state.Profiles {
		nextState.Profiles[id] = p
	}

	// Update the profile with new preferences
	profile := nextState.Profiles[s.state.ActiveProfileID]
	profile.UIPreferences = uiPrefs
	nextState.Profiles[s.state.ActiveProfileID] = profile

	validatedState, err := types.ValidateState(nextState)
	if err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Invalid UI preferences"),
			Profile:         s.state.Profiles[s.state.ActiveProfileID],
			Errors: []types.UserProfilesError{
				userProfilesError(s.state.ActiveProfileID, "", "", types.ErrorUnknown, "", "Invalid UI preferences: "+err.Error()),
			},
		}
	}

	s.setState(validatedState)

	if err := WriteUserProfilesState(validatedState); err != nil {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Failed to persist UI preferences"),
			Profile:         validatedState.Profiles[s.state.ActiveProfileID],
			Errors: []types.UserProfilesError{
				userProfilesError(s.state.ActiveProfileID, "", "", types.ErrorPersistFailed, "", "Failed to persist UI preferences: "+err.Error()),
			},
		}
	}

	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("UI preferences updated"),
		Profile:         validatedState.Profiles[s.state.ActiveProfileID],
		Errors:          []types.UserProfilesError{},
	}
}

// ===== Profile Management ===== //

// CreateProfile creates a new profile with the provided name and preferences, ensuring that the name is unique and non-empty, and persists the updated profiles state.
func (s *UserProfiles) CreateProfile(req types.CreateProfileRequest) types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("CreateProfile")

	// trim whitespace and validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return profileStateErrorResult(
			"Invalid profile name",
			types.UserProfile{},
			userProfilesError("", "", "", types.ErrorInvalidProfileName, "", "Profile name is required"),
		)
	}

	// enforce unique profile names (case-insensitive)
	for _, profile := range s.state.Profiles {
		if strings.EqualFold(strings.TrimSpace(profile.Name), name) {
			return profileStateErrorResult(
				"Profile name already exists",
				types.UserProfile{},
				userProfilesError("", "", "", types.ErrorDuplicateName, "", fmt.Sprintf("Profile name %q already exists", name)),
			)
		}
	}

	nextState := copyProfilesState(s.state)
	profile := buildProfileFromRequest(req)
	// Generate a unique (local-only) profile ID as well as a global UUID
	profile.ID = nextGeneratedProfileID(nextState.Profiles)
	profile.UUID = types.DefaultProfile().UUID
	profile.Name = name

	nextState.Profiles[profile.ID] = profile
	validatedState, validationErrResult := s.validateAndPersistProfileState(nextState, "creation", profile.ID, types.UserProfile{})
	if validationErrResult != nil {
		return *validationErrResult
	}

	s.setState(validatedState)
	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("Profile created"),
		Profile:         validatedState.Profiles[profile.ID],
		Errors:          []types.UserProfilesError{},
	}
}

func (s *UserProfiles) DeleteProfile(profileID string) types.UserProfileResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRequest("DeleteProfile", "profile_id", profileID)

	profileID = strings.TrimSpace(profileID)
	profile, exists := s.state.Profiles[profileID]
	if !exists {
		err := profileNotFoundError(profileID)
		return profileStateErrorResult("Profile not found", types.UserProfile{}, err)
	}
	if profileID == types.DefaultProfileID {
		return profileStateErrorResult(
			"Cannot delete default profile",
			s.state.Profiles[s.state.ActiveProfileID],
			userProfilesError(profileID, "", "", types.ErrorDefaultProtected, "", "Cannot delete default profile"),
		)
	}
	if profileID == s.state.ActiveProfileID {
		return profileStateErrorResult(
			"Cannot delete active profile",
			s.state.Profiles[s.state.ActiveProfileID],
			userProfilesError(profileID, "", "", types.ErrorActiveProtected, "", "Cannot delete active profile"),
		)
	}

	nextState := copyProfilesState(s.state)
	delete(nextState.Profiles, profileID)

	validatedState, validationErrResult := s.validateAndPersistProfileState(nextState, "deletion", profileID, s.state.Profiles[s.state.ActiveProfileID])
	if validationErrResult != nil {
		return *validationErrResult
	}

	s.setState(validatedState)
	archivePath := profileArchivePath(profile.UUID)
	if removeErr := os.Remove(archivePath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return profileStateErrorResult(
			"Profile deleted but failed to remove archive",
			validatedState.Profiles[validatedState.ActiveProfileID],
			userProfilesError(profileID, "", "", types.ErrorArchiveUpdate, "", "Profile deleted but failed to remove archive: "+removeErr.Error()),
		)
	}

	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("Profile deleted"),
		Profile:         validatedState.Profiles[validatedState.ActiveProfileID],
		Errors:          []types.UserProfilesError{},
	}
}

// SwapProfile attempts to swap the active profile to the target profile, returning as a no-op if the target profile is already active
func (s *UserProfiles) SwapProfile(req types.SwapProfileRequest) types.UserProfileResult {
	s.logRequest("SwapProfile", "profile_id", req.ProfileID, "force_swap", req.ForceSwap)

	currentProfile, targetProfile, validationResult, ok := s.swapProfilesSnapshot(req.ProfileID)
	if !ok {
		return validationResult
	}

	// If already active, no-op with success.
	if currentProfile.ID == targetProfile.ID {
		s.Logger.Info("Target profile is already active, no swap needed", "profile_id", targetProfile.ID)
		return types.UserProfileResult{
			GenericResponse: types.SuccessResponse("Profile already active"),
			Profile:         targetProfile,
			Errors:          []types.UserProfilesError{},
		}
	}

	// Validate current profile archive status
	isCurrentArchiveFresh, currentErrResult := s.resolveProfileArchiveFreshness(currentProfile, currentProfile, "current")
	if currentErrResult != nil {
		return *currentErrResult
	}
	// Create a new archive for the current profile before swapping, unless the archive is already fresh, to ensure that there is a backup for the profile in case we need to restore it in the future
	if !isCurrentArchiveFresh {
		archiveResult := s.CreateProfileArchive(currentProfile.ID)
		if archiveResult.Status == types.ResponseError {
			s.Logger.Error("Failed to update current profile archive before swap", errors.New(archiveResult.Message), "profile_id", currentProfile.ID)
			return profileStateErrorResult(
				"Failed to update current profile archive before swap",
				currentProfile,
				userProfilesError(currentProfile.ID, "", "", types.ErrorArchiveUpdate, "", "Failed to update current profile archive before swap: "+archiveResult.Message),
			)
		}
	}
	// Validate target profile archive status
	isTargetArchiveFresh, targetErrResult := s.resolveProfileArchiveFreshness(targetProfile, currentProfile, "target")
	if targetErrResult != nil {
		return *targetErrResult
	}
	// If the target archive is not fresh, request a confirmation from the frontend before proceeding with the swap, as swapping to a profile with a stale or missing archive would result in potential download and sync operations post-swap
	if !isTargetArchiveFresh && !req.ForceSwap {
		errorType := types.ErrorArchiveStale
		exists, statErr := profileArchiveExists(targetProfile.UUID)
		if statErr == nil && !exists {
			errorType = types.ErrorArchiveMissing
		}
		s.Logger.Warn("Target profile archive is missing or stale, confirming with user before swap", "profile_id", targetProfile.ID, "archive_exists", exists, "stat_error", statErr)
		return types.UserProfileResult{
			GenericResponse: types.WarnResponse("Target profile archive is missing or stale; confirm swap to continue"),
			Profile:         currentProfile,
			Errors: []types.UserProfilesError{
				userProfilesError(
					targetProfile.ID,
					"",
					"",
					errorType,
					"",
					fmt.Sprintf("Archive for profile %q is missing or stale", targetProfile.ID),
				),
			},
		}
	}

	// Proceed with the swap once we determine that both the current and target profile archives are in a known state (and after confirming with the user if the target archive is not fresh)
	swappedProfile, persistErr := s.setActiveProfile(targetProfile.ID)
	if persistErr != nil {
		s.Logger.Error("Failed to persist active profile swap", persistErr, "target_profile_id", targetProfile.ID)
		return profileStateErrorResult(
			"Failed to persist active profile swap",
			currentProfile,
			userProfilesError(req.ProfileID, "", "", types.ErrorPersistFailed, "", "Failed to persist active profile swap: "+persistErr.Error()),
		)
	}

	// If the target archive is fresh, we can skip the restore/sync steps
	if isTargetArchiveFresh {
		restoreResult := s.RestoreProfileArchive(targetProfile.ID)
		if restoreResult.Status == types.ResponseError {
			s.Logger.Error("Active profile changed, but failed to restore from archive", errors.New(restoreResult.Message), "profile_id", targetProfile.ID)
			return profileStateErrorResult(
				"Active profile changed, but archive restoration failed",
				swappedProfile,
				userProfilesError(req.ProfileID, "", "", types.ErrorArchiveRestore, "", "Active profile changed, but archive restoration failed: "+restoreResult.Message),
			)
		}
		return types.UserProfileResult{
			GenericResponse: types.SuccessResponse("Profile swapped successfully"),
			Profile:         swappedProfile,
			Errors:          []types.UserProfilesError{},
		}
	}

	// Otherwise, if the target archive is not fresh, we will need to run a new subscriptions sync to ensure that the profile is up to date post-swap
	syncResult := s.SyncSubscriptions(targetProfile.ID, false, false)
	if syncResult.Status == types.ResponseError {
		return types.UserProfileResult{
			GenericResponse: types.ErrorResponse("Profile swapped, but failed to sync subscriptions"),
			Profile:         swappedProfile,
			Errors:          syncResult.Errors,
		}
	}
	if syncResult.Status == types.ResponseWarn {
		return types.UserProfileResult{
			GenericResponse: types.WarnResponse("Profile swapped with subscription sync warnings"),
			Profile:         swappedProfile,
			Errors:          syncResult.Errors,
		}
	}

	s.Logger.Info("Profile swapped successfully", "profile_id", targetProfile.ID)
	return types.UserProfileResult{
		GenericResponse: types.SuccessResponse("Profile swapped successfully"),
		Profile:         swappedProfile,
		Errors:          []types.UserProfilesError{},
	}
}

// ===== Profile Management Helpers ===== //

// validateAndPersistProfileState validates the provided profile state and persists it to disk if valid, returning any errors encountered during validation or persistence
func (s *UserProfiles) validateAndPersistProfileState(
	nextState types.UserProfilesState,
	action string,
	profileID string,
	profile types.UserProfile,
) (types.UserProfilesState, *types.UserProfileResult) {
	validationMessage := fmt.Sprintf("Invalid profile state after %s", action)
	validatedState, err := types.ValidateState(nextState)
	if err != nil {
		s.Logger.Error(validationMessage, err, "active_profile_id", s.state.ActiveProfileID)
		result := profileStateErrorResult(
			validationMessage,
			profile,
			userProfilesError(profileID, "", "", types.ErrorUnknown, "", validationMessage+": "+err.Error()),
		)
		return types.UserProfilesState{}, &result
	}

	persistMessage := fmt.Sprintf("Failed to persist profile %s", action)
	if err := WriteUserProfilesState(validatedState); err != nil {
		s.Logger.Error(persistMessage, err, "profile_id", profileID)
		result := profileStateErrorResult(
			persistMessage,
			profile,
			userProfilesError(profileID, "", "", types.ErrorPersistFailed, "", persistMessage+": "+err.Error()),
		)
		return types.UserProfilesState{}, &result
	}

	return validatedState, nil
}

func (s *UserProfiles) resolveProfileArchiveFreshness(
	profile types.UserProfile,
	resultProfile types.UserProfile,
	kind string,
) (bool, *types.UserProfileResult) {
	isFresh, archiveErr := s.isProfileArchiveFresh(profile)
	if archiveErr == nil {
		return isFresh, nil
	}

	message := fmt.Sprintf("Failed to inspect %s profile archive", kind)
	s.Logger.Error(message, archiveErr, "profile_id", profile.ID)
	result := profileStateErrorResult(
		message,
		resultProfile,
		userProfilesError(profile.ID, "", "", types.ErrorArchiveUpdate, "", message+": "+archiveErr.Error()),
	)
	return false, &result
}

// buildProfileFromRequest constructs a UserProfile from a CreateProfileRequest, applying defaults for any missing fields.
// It does not perform validation, so the resulting UserProfile may be invalid.
func buildProfileFromRequest(req types.CreateProfileRequest) types.UserProfile {
	profile := types.DefaultProfile()
	if req.UIPreferences != nil {
		profile.UIPreferences = *req.UIPreferences
	}
	if req.SystemPreferences != nil {
		profile.SystemPreferences = *req.SystemPreferences
	}
	if req.Favorites != nil {
		favorites := profile.Favorites
		if req.Favorites.Authors != nil {
			favorites.Authors = append([]string{}, req.Favorites.Authors...)
		}
		if req.Favorites.Maps != nil {
			favorites.Maps = append([]string{}, req.Favorites.Maps...)
		}
		if req.Favorites.Mods != nil {
			favorites.Mods = append([]string{}, req.Favorites.Mods...)
		}
		profile.Favorites = favorites
	}
	if req.Subscriptions != nil {
		subscriptions := profile.Subscriptions
		if req.Subscriptions.Maps != nil {
			subscriptions.Maps = utils.CloneStringMap(req.Subscriptions.Maps)
		}
		if req.Subscriptions.LocalMaps != nil {
			subscriptions.LocalMaps = utils.CloneStringMap(req.Subscriptions.LocalMaps)
		}
		if req.Subscriptions.Mods != nil {
			subscriptions.Mods = utils.CloneStringMap(req.Subscriptions.Mods)
		}
		profile.Subscriptions = subscriptions
	}
	return profile
}

// nextGeneratedProfileID generates a new profile ID in the format "profile_X" where X is the next available integer based on existing profile IDs.
// This is intended to ensure unique profile IDs for any local profile excluding the default profile.
// The function ignores any profile IDs that do not match the expected format.
func nextGeneratedProfileID(profiles map[string]types.UserProfile) string {
	maxID := -1
	for profileID := range profiles {
		if !strings.HasPrefix(profileID, "profile_") {
			continue
		}
		suffix := strings.TrimPrefix(profileID, "profile_")
		value, err := strconv.Atoi(suffix)
		if err != nil {
			continue
		}
		if value > maxID {
			maxID = value
		}
	}
	return fmt.Sprintf("profile_%d", maxID+1)
}

func (s *UserProfiles) swapProfilesSnapshot(profileID string) (types.UserProfile, types.UserProfile, types.UserProfileResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profileID = strings.TrimSpace(profileID)
	// currentProfile should always resolve successfully
	currentProfile, _ := profileSnapshotFromState(s.state, s.state.ActiveProfileID)
	targetProfile, targetErr := profileSnapshotFromState(s.state, profileID)
	if targetErr != nil {
		err := profileNotFoundError(profileID)
		result := profileStateErrorResult("Profile not found", currentProfile, err)
		return types.UserProfile{}, types.UserProfile{}, result, false
	}

	return currentProfile, targetProfile, types.UserProfileResult{}, true
}

// setActiveProfile updates the active profile in state and persists it to disk.
func (s *UserProfiles) setActiveProfile(profileID string) (types.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state.Profiles[profileID]; !exists {
		return types.UserProfile{}, profileNotFoundError(profileID)
	}
	nextState := copyProfilesState(s.state)
	nextState.ActiveProfileID = profileID
	validatedState, err := types.ValidateState(nextState)
	if err != nil {
		return types.UserProfile{}, err
	}
	if err := WriteUserProfilesState(validatedState); err != nil {
		return types.UserProfile{}, err
	}
	s.setState(validatedState)
	return validatedState.Profiles[profileID], nil
}
