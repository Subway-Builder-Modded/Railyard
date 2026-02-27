package main

import (
	"errors"
	"fmt"
	"sync"

	"railyard/internal/files"
	"railyard/internal/types"
)

type UserProfiles struct {
	state  types.UserProfilesState
	logger Logger
	mu     sync.Mutex
	loaded bool
}

var ErrProfilesNotLoaded = errors.New("profiles state not loaded")

func NewUserProfiles(logger Logger) *UserProfiles {
	return &UserProfiles{
		logger: logger,
	}
}

func (s *UserProfiles) setState(state types.UserProfilesState) {
	s.state = state
	s.loaded = true
}

func writeUserProfilesState(state types.UserProfilesState) error {
	return files.WriteJSON(UserProfilesPath(), "user profiles", state)
}

func readUserProfilesState() (types.UserProfilesState, error) {
	return files.ReadJSON[types.UserProfilesState](
		UserProfilesPath(),
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
	if s.loaded {
		return s.resolveActiveProfile()
	}

	state, err := readUserProfilesState()
	if err != nil {
		return types.UserProfile{}, err
	}

	// If no profiles exist on disk, initialize with default profile
	if len(state.Profiles) == 0 {
		bootstrapped := types.InitialProfilesState()
		if err := writeUserProfilesState(bootstrapped); err != nil {
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
	defaultState := types.InitialProfilesState()
	s.setState(defaultState)
	return writeUserProfilesState(defaultState)
}

// quarantineUserProfiles moves the user profiles file to a "quarantined" path in the same directory
// If the source file is missing, it is treated as a no-op.
func (s *UserProfiles) quarantineUserProfiles() (success bool, backupPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return QuarantineFile(UserProfilesPath(), s.logger)
}

// ResolveActiveProfile returns the active profile from loaded in-memory state.
// Callers must ensure LoadProfiles has completed successfully first.
func (s *UserProfiles) ResolveActiveProfile() (activeProfile types.UserProfile, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveActiveProfile()
}

func (s *UserProfiles) resolveActiveProfile() (activeProfile types.UserProfile, err error) {
	if !s.loaded {
		return types.UserProfile{}, ErrProfilesNotLoaded
	}

	profile, ok := s.state.Profiles[s.state.ActiveProfileID]
	if !ok {
		return types.UserProfile{}, fmt.Errorf("active profile %q missing from loaded state", s.state.ActiveProfileID)
	}

	return profile, nil
}
