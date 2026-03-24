package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/mem"
)

// ThemeMode represents the UI theme preference of a user profile.
type ThemeMode string

const (
	ThemeDark      ThemeMode = "dark"
	ThemeDarkLow   ThemeMode = "dark_low"
	ThemeDarkHigh  ThemeMode = "dark_high"
	ThemeLight     ThemeMode = "light"
	ThemeLightLow  ThemeMode = "light_low"
	ThemeLightHigh ThemeMode = "light_high"
	ThemeSystem    ThemeMode = "system"
)

// PageSize represents the number of entries to display per page when the user browses the registry.
type PageSize int

const (
	PageSize12 PageSize = 12
	PageSize24 PageSize = 24
	PageSize48 PageSize = 48
)

// SearchViewMode represents the display mode for search results in the Browse menu in the UI.
type SearchViewMode string

const (
	SearchViewModeList    SearchViewMode = "list"
	SearchViewModeFull    SearchViewMode = "full"    // full-size cards with tags
	SearchViewModeCompact SearchViewMode = "compact" // smaller-size cards without tags
)

// UIPreferences represents user preferences related to application UI/UX.
type UIPreferences struct {
	Theme          ThemeMode      `json:"theme"`
	DefaultPerPage PageSize       `json:"defaultPerPage"`
	SearchViewMode SearchViewMode `json:"searchViewMode"`
}

// SystemPreferences represents user preferences related to application behavior and features.
type SystemPreferences struct {
	RefreshRegistryOnStartup bool `json:"refreshRegistryOnStartup"` // Whether to refresh the registry on application startup
	// AutoUpdateSubscriptions  bool `json:"autoUpdateSubscriptions"`  // Whether to automatically update subscribed maps/mods when new versions are released
	ExtraHeapSize int  `json:"extraMemorySize,omitempty"` // Amount of memory to allocate to the game, if specified by the user (in MB). This is an optional preference that can help improve performance for users with larger libraries or lower-end hardware.
	UseDevTools   bool `json:"useDevTools,omitempty"`     // Whether to enable developer tools for the game on launch.
}

// Favorites represents favorite authors/maps/mods for a profile.
type Favorites struct {
	Authors []string `json:"authors"`
	Maps    []string `json:"maps"`
	Mods    []string `json:"mods"`
}

// Subscriptions represents the maps/mods and their respective versions that a user is subscribed to.
type Subscriptions struct {
	Maps      map[string]string `json:"maps"`
	LocalMaps map[string]string `json:"localMaps"`
	Mods      map[string]string `json:"mods"`
}

type SubscriptionAction string

const (
	SubscriptionActionSubscribe   SubscriptionAction = "subscribe"
	SubscriptionActionUnsubscribe SubscriptionAction = "unsubscribe"
)

func IsValidSubscriptionAction(action SubscriptionAction) bool {
	switch action {
	case SubscriptionActionSubscribe, SubscriptionActionUnsubscribe:
		return true
	default:
		return false
	}
}

type SubscriptionUpdateItem struct {
	Version Version   `json:"version"`
	Type    AssetType `json:"type"`
	IsLocal bool      `json:"isLocal,omitempty"`
}

type UpdateSubscriptionsRequest struct {
	ProfileID         string                            `json:"profileId"`
	Assets            map[string]SubscriptionUpdateItem `json:"assets"`
	Action            SubscriptionAction                `json:"action"`
	ForceSync         bool                              `json:"forceSync"`
	ReplaceOnConflict bool                              `json:"replaceOnConflict"`
}

type ImportAssetRequest struct {
	ProfileID         string    `json:"profileId"`
	AssetType         AssetType `json:"assetType"`
	ZipPath           string    `json:"zipPath"`
	ReplaceOnConflict bool      `json:"replaceOnConflict"`
}

type UpdateSubscriptionsToLatestRequest struct {
	ProfileID string                     `json:"profileId"`
	Apply     bool                       `json:"apply"`
	Targets   []SubscriptionUpdateTarget `json:"targets,omitempty"`
}

type UpdateSubscriptionRequestType string

const (
	UpdateSubscriptions UpdateSubscriptionRequestType = "update_subscriptions"
	LatestCheck         UpdateSubscriptionRequestType = "latest_check"
	LatestApply         UpdateSubscriptionRequestType = "latest_apply"
	ImportAsset         UpdateSubscriptionRequestType = "import_asset"
)

type SubscriptionOperation struct {
	AssetID string             `json:"assetId"`
	Type    AssetType          `json:"type"`
	Action  SubscriptionAction `json:"action"`
	Version Version            `json:"version"`
}

type SubscriptionUpdateTarget struct {
	AssetID string    `json:"assetId"`
	Type    AssetType `json:"type"`
}

type PendingSubscriptionUpdate struct {
	AssetID        string    `json:"assetId"`
	Type           AssetType `json:"type"`
	CurrentVersion Version   `json:"currentVersion"`
	LatestVersion  Version   `json:"latestVersion"`
}

type UserProfilesErrorType string

const (
	ErrorProfileNotFound   UserProfilesErrorType = "profile_not_found"
	ErrorProfilesNotLoaded UserProfilesErrorType = "profiles_not_loaded"
	ErrorInvalidAssetID    UserProfilesErrorType = "invalid_asset_id"
	ErrorInvalidAssetType  UserProfilesErrorType = "invalid_asset_type"
	ErrorInvalidVersion    UserProfilesErrorType = "invalid_version"
	ErrorInvalidAction     UserProfilesErrorType = "invalid_action"
	ErrorPersistFailed     UserProfilesErrorType = "persist_failed"
	ErrorSyncFailed        UserProfilesErrorType = "sync_failed"
	ErrorSyncSuperseded    UserProfilesErrorType = "sync_superseded"
	ErrorLookupFailed      UserProfilesErrorType = "lookup_failed"
	ErrorUnknown           UserProfilesErrorType = "unknown"
)

type UserProfilesError struct {
	ProfileID           string                `json:"profileId"`
	AssetID             string                `json:"assetId"`
	AssetType           AssetType             `json:"assetType"`
	ErrorType           UserProfilesErrorType `json:"errorType"`
	DownloaderErrorType DownloaderErrorType   `json:"downloaderErrorType,omitempty"`
	Message             string                `json:"message"`
}

func (e UserProfilesError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return string(e.ErrorType)
}

type UserProfileResult struct {
	GenericResponse
	Profile UserProfile         `json:"profile"`
	Errors  []UserProfilesError `json:"errors"`
}

type UpdateSubscriptionsResult struct {
	GenericResponse
	RequestType    UpdateSubscriptionRequestType `json:"requestType"`
	HasUpdates     bool                          `json:"hasUpdates"`     // whether there are updates to apply based on the profile's subscriptions and the registry state at the time of the request
	PendingCount   int                           `json:"pendingCount"`   // number of pending updates (only relevant for latest_check and latest_apply request types)
	PendingUpdates []PendingSubscriptionUpdate   `json:"pendingUpdates"` // list of pending updates with details (if any)
	Applied        bool                          `json:"applied"`        // whether the updates were applied (or if this request was just a check)
	Profile        UserProfile                   `json:"profile"`        // the updated profile after applying subscription changes (if applicable)
	Persisted      bool                          `json:"persisted"`      // whether the updated profile state was persisted to disk (if applicable)
	Operations     []SubscriptionOperation       `json:"operations"`     // the list of subscription operations that were performed (if any)
	Errors         []UserProfilesError           `json:"errors"`         // errors that were encountered during the subscription update process
	Conflicts      []MapCodeConflict             `json:"conflicts"`      // map code conflicts that require explicit confirmation to replace
}

type SyncSubscriptionsResult struct {
	GenericResponse
	ProfileID  string                  `json:"profileId"`
	Operations []SubscriptionOperation `json:"operations"` // the list of subscription operations that were performed (if any)
	Errors     []UserProfilesError     `json:"errors"`     // errors that were encountered during the subscription sync process
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
		SearchViewMode: SearchViewModeFull,
	}
}

func defaultSystemPreferences() SystemPreferences {
	return SystemPreferences{
		RefreshRegistryOnStartup: true,
		ExtraHeapSize:            -1,
		UseDevTools:              false,
		// AutoUpdateSubscriptions:  false,
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
		Maps:      map[string]string{},
		LocalMaps: map[string]string{},
		Mods:      map[string]string{},
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
	case ThemeDark,
		ThemeDarkLow,
		ThemeDarkHigh,
		ThemeLight,
		ThemeLightLow,
		ThemeLightHigh,
		ThemeSystem:
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

func isValidSearchViewMode(value SearchViewMode) bool {
	switch value {
	case SearchViewModeList, SearchViewModeFull, SearchViewModeCompact:
		return true
	default:
		return false
	}
}

func normalizeUIPreferences(prefs UIPreferences) UIPreferences {
	// bootstrap newly added fields with defaults if missing to avoid issues with older versions of profiles state
	if prefs.SearchViewMode == "" {
		prefs.SearchViewMode = SearchViewModeFull
	}
	return prefs
}

func areValidUIPreferences(prefs UIPreferences) bool {
	prefs = normalizeUIPreferences(prefs)
	return isValidTheme(prefs.Theme) && isValidPageSize(prefs.DefaultPerPage) && isValidSearchViewMode(prefs.SearchViewMode)
}

func areValidSystemPreferences(prefs SystemPreferences) bool {
	ramMax, err := mem.VirtualMemory()
	maxVal := int(ramMax.Total / (1024 * 1024)) // convert to MB
	if prefs.ExtraHeapSize < -1 {
		return false
	}

	// Not bracketed to 5/8ths, developers and those who know what they're doing can manually set it higher
	if err == nil && prefs.ExtraHeapSize >= int(maxVal) {
		return false
	}
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
		p.UIPreferences = normalizeUIPreferences(p.UIPreferences)
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
		if p.Subscriptions.LocalMaps == nil {
			p.Subscriptions.LocalMaps = map[string]string{}
		}
		if p.Favorites.Authors == nil || p.Favorites.Maps == nil || p.Favorites.Mods == nil {
			return UserProfilesState{}, fmt.Errorf("%w: profiles[%q] favorites authors/maps/mods must be present", ErrMalformedProfile, key)
		}

		s.Profiles[key] = p
	}

	return s, nil
}

var (
	ErrInvalidState         = errors.New("invalid profiles state")
	ErrMismatchedProfileKey = errors.New("mismatched profile key")
	ErrMalformedProfile     = errors.New("malformed profile")
)
