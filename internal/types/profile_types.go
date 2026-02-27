package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ThemeMode represents the UI theme preference of a user profile.
type ThemeMode string

const (
	ThemeDark   ThemeMode = "dark"
	ThemeLight  ThemeMode = "light"
	ThemeSystem ThemeMode = "system"
)

// PageSize represents the number of entries to display per page when the user browses the registry.
type PageSize int

const (
	PageSize12 PageSize = 12
	PageSize24 PageSize = 24
	PageSize48 PageSize = 48
)

// UIPreferences represents user preferences related to application UI/UX.
type UIPreferences struct {
	Theme          ThemeMode `json:"theme"`
	DefaultPerPage PageSize  `json:"defaultPerPage"`
}

// SystemPreferences represents user preferences related to application behavior and features.
type SystemPreferences struct {
	RefreshRegistryOnStartup bool `json:"refreshRegistryOnStartup"` // Whether to refresh the registry on application startup
	AutoUpdateSubscriptions  bool `json:"autoUpdateSubscriptions"`  // Whether to automatically update subscribed maps/mods when new versions are released
}

// Favorites represents favorite authors/maps/mods for a profile.
type Favorites struct {
	Authors []string `json:"authors"`
	Maps    []string `json:"maps"`
	Mods    []string `json:"mods"`
}

// Subscriptions represents the maps/mods and their respective versions that a user is subscribed to.
type Subscriptions struct {
	Maps map[string]string `json:"maps"`
	Mods map[string]string `json:"mods"`
}

// UserProfile represents a profile within the application.
type UserProfile struct {
	ID                string            `json:"id"`
	UUID              string            `json:"uuid"`
	Name              string            `json:"name"`
	UIPreferences     UIPreferences     `json:"uiPreferences"`
	SystemPreferences SystemPreferences `json:"systemPreferences"`
	Subscriptions     Subscriptions     `json:"subscriptions"`
	Favorites         Favorites         `json:"favorites"`
}

// UserProfilesState is the state persisted on disk.
type UserProfilesState struct {
	ActiveProfileID string                 `json:"activeProfileId"`
	Profiles        map[string]UserProfile `json:"profiles"`
}

// ===== Defaults =====

const DefaultProfileID = "__default__"
const DefaultProfileName = "Default"


func defaultUIPreferences() UIPreferences {
	return UIPreferences{
		Theme:          ThemeDark,
		DefaultPerPage: PageSize12,
	}
}

func defaultSystemPreferences() SystemPreferences {
	return SystemPreferences{
		RefreshRegistryOnStartup: true,
		AutoUpdateSubscriptions:  false,
	}
}

func defaultFavorites() Favorites {
	return Favorites{
		Authors: []string{},
		Maps:    []string{},
		Mods:    []string{},
	}
}

func defaultSubscriptions() Subscriptions {
	return Subscriptions{
		Maps: map[string]string{},
		Mods: map[string]string{},
	}
}

func DefaultProfile() UserProfile {
	return UserProfile{
		ID:                DefaultProfileID,
		UUID:              uuid.NewString(),
		Name:              DefaultProfileName,
		UIPreferences:     defaultUIPreferences(),
		SystemPreferences: defaultSystemPreferences(),
		Subscriptions:     defaultSubscriptions(),
		Favorites:         defaultFavorites(),
	}
}

func InitialProfilesState() UserProfilesState {
	return UserProfilesState{
		ActiveProfileID: DefaultProfileID,
		Profiles: map[string]UserProfile{
			DefaultProfileID: DefaultProfile(),
		},
	}
}

func isValidTheme(theme ThemeMode) bool {
	switch theme {
	case ThemeDark, ThemeLight, ThemeSystem:
		return true
	default:
		return false
	}
}

func isValidPageSize(value PageSize) bool {
	switch value {
	case PageSize12, PageSize24, PageSize48:
		return true
	default:
		return false
	}
}

func areValidUIPreferences(prefs UIPreferences) bool {
	return isValidTheme(prefs.Theme) && isValidPageSize(prefs.DefaultPerPage)
}

func areValidSystemPreferences(prefs SystemPreferences) bool {
	// No validation rules for system preferences given all fields are boolean and will default to false if missing on parse
	return true
}

// ValidateState checks that the loaded state from disk is not malformed. 
// Railyard should control writes to this file, so any malformed state would be indicative of a bug or manual tampering
func ValidateState(s UserProfilesState) (UserProfilesState, error) {
	// Default profile must exist.
	if _, ok := s.Profiles[DefaultProfileID]; !ok {
		return UserProfilesState{}, fmt.Errorf("%w: default profile %q missing", ErrInvalidState, DefaultProfileID)
	}
	// Active profile must exist.
	if _, ok := s.Profiles[s.ActiveProfileID]; !ok {
		return UserProfilesState{}, fmt.Errorf("%w: active profile %q missing", ErrInvalidState, s.ActiveProfileID)
	}

	for key, p := range s.Profiles {
		// Key must match ID.
		if p.ID != key {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q].id=%q", ErrMismatchedProfileKey, key, p.ID)
		}
		if strings.TrimSpace(p.ID) == "" ||
			strings.TrimSpace(p.Name) == "" {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] requires non-empty id/uuid/name", ErrMalformedProfile, key)
		}
		if _, err := uuid.Parse(p.UUID); err != nil {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] requires non-empty id/uuid/name", ErrMalformedProfile, key)
		}

		// Preferences must be valid.
		if !areValidUIPreferences(p.UIPreferences) {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] has invalid UI preferences", ErrMalformedProfile, key)
		}

		if !areValidSystemPreferences(p.SystemPreferences) {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] has invalid system preferences", ErrMalformedProfile, key)
		}

		// Collections must be present (non-nil).
		if p.Subscriptions.Maps == nil || p.Subscriptions.Mods == nil {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] subscriptions maps/mods must be present", ErrMalformedProfile, key)
		}
		if p.Favorites.Authors == nil || p.Favorites.Maps == nil || p.Favorites.Mods == nil {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] favorites authors/maps/mods must be present", ErrMalformedProfile, key)
		}
	}

	return s, nil
}

var (
	ErrInvalidState         = errors.New("invalid profiles state")
	ErrMismatchedProfileKey = errors.New("mismatched profile key")
	ErrMalformedProfile				= errors.New("malformed profile")
)
