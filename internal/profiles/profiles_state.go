package profiles

import (
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/types"
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

// TODO: Add functions to Create/Delete/Swap profiles
