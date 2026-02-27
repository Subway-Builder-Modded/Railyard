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
	mu     sync.Mutex
	loaded bool
}

var ErrProfilesNotLoaded = errors.New("profiles state not loaded")

func NewUserProfiles() *UserProfiles {
	return &UserProfiles{}
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
func (s *UserProfiles) LoadProfiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}

	state, err := readUserProfilesState()
	if err != nil {
		return err
	}

	// If no profiles exist on disk, initialize with default profile
	if len(state.Profiles) == 0 {
		bootstrapped := types.InitialProfilesState()
		if err := writeUserProfilesState(bootstrapped); err != nil {
			return err
		}
		s.setState(bootstrapped)
		return nil
	}

	validatedState, err := types.ValidateState(state)
	if err != nil {
		return err
	}

	s.setState(validatedState)
	return nil
}

func (s *UserProfiles) GetActiveProfile() types.UserProfile {
	profile, _ := s.ResolveActiveProfile()
	return profile
}

// ResolveActiveProfile returns the active profile from loaded in-memory state.
// Callers must ensure LoadProfiles has completed successfully first.
func (s *UserProfiles) ResolveActiveProfile() (types.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.loaded {
		return types.UserProfile{}, ErrProfilesNotLoaded
	}

	profile, ok := s.state.Profiles[s.state.ActiveProfileID]
	if !ok {
		return types.UserProfile{}, fmt.Errorf("active profile %q missing from loaded state", s.state.ActiveProfileID)
	}

	return profile, nil
}
