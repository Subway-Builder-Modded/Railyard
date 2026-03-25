package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidTheme(t *testing.T) {
	require.True(t, isValidTheme(ThemeDark))
	require.True(t, isValidTheme(ThemeLight))
	require.True(t, isValidTheme(ThemeSystem))
	require.False(t, isValidTheme(ThemeMode("custom")))
}

func TestIsValidPageSize(t *testing.T) {
	require.True(t, isValidPageSize(PageSize12))
	require.True(t, isValidPageSize(PageSize24))
	require.True(t, isValidPageSize(PageSize48))
	require.False(t, isValidPageSize(PageSize(0)))
}

func TestAreValidUIPreferences(t *testing.T) {
	require.True(t, areValidUIPreferences(UIPreferences{
		Theme:          ThemeDark,
		DefaultPerPage: PageSize12,
		SearchViewMode: SearchViewModeFull,
	}))
	require.True(t, areValidUIPreferences(UIPreferences{
		Theme:          ThemeLight,
		DefaultPerPage: PageSize24,
		SearchViewMode: "",
	}))
	require.False(t, areValidUIPreferences(UIPreferences{
		Theme:          ThemeMode("custom"),
		DefaultPerPage: PageSize12,
		SearchViewMode: SearchViewModeCompact,
	}))
	require.False(t, areValidUIPreferences(UIPreferences{
		Theme:          ThemeDark,
		DefaultPerPage: PageSize(999),
		SearchViewMode: SearchViewModeList,
	}))
	require.False(t, areValidUIPreferences(UIPreferences{
		Theme:          ThemeLight,
		DefaultPerPage: PageSize24,
		SearchViewMode: SearchViewMode("unknown"),
	}))
}

func TestAreValidSystemPreferences(t *testing.T) {
	require.True(t, areValidSystemPreferences(SystemPreferences{}))
	require.True(t, areValidSystemPreferences(SystemPreferences{
		RefreshRegistryOnStartup: true,
		// AutoUpdateSubscriptions:  true,
	}))
}

func TestValidateStateAcceptsInitialProfilesState(t *testing.T) {
	state := InitialProfilesState()

	validated, err := ValidateState(state)
	require.NoError(t, err)
	require.Equal(t, state, validated)
}

func TestValidateStateBackfillsEmptySearchViewMode(t *testing.T) {
	state := InitialProfilesState()
	profile := state.Profiles[DefaultProfileID]
	profile.UIPreferences.SearchViewMode = ""
	state.Profiles[DefaultProfileID] = profile

	validated, err := ValidateState(state)
	require.NoError(t, err)
	require.Equal(t, SearchViewModeFull, validated.Profiles[DefaultProfileID].UIPreferences.SearchViewMode)
}

func assertInvalidState(t *testing.T, mutate func(*UserProfilesState), exErr error) {
	t.Helper()

	state := InitialProfilesState()
	mutate(&state)

	_, err := ValidateState(state)
	require.Error(t, err)
	require.ErrorIs(t, err, exErr)
}

func TestValidateStateRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*UserProfilesState)
		exErr  error
	}{
		{
			name: "missing active profile",
			mutate: func(state *UserProfilesState) {
				state.ActiveProfileID = "missing"
			},
			exErr: ErrInvalidState,
		},
		{
			name: "missing default profile",
			mutate: func(state *UserProfilesState) {
				delete(state.Profiles, DefaultProfileID)
			},
			exErr: ErrInvalidState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInvalidState(t, tt.mutate, tt.exErr)
		})
	}
}

func assertInvalidProfile(t *testing.T, mutate func(*UserProfile), exErr error) {
	t.Helper()

	state := InitialProfilesState()
	profile := state.Profiles[DefaultProfileID]
	mutate(&profile)
	state.Profiles[DefaultProfileID] = profile

	_, err := ValidateState(state)
	require.Error(t, err)
	require.ErrorIs(t, err, exErr)
}

func TestValidateStateRejectsInvalidProfile(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*UserProfile)
		exErr  error
	}{
		{
			name: "mismatched profile key and id",
			mutate: func(profile *UserProfile) {
				profile.ID = "not_default"
			},
			exErr: ErrMismatchedProfileKey,
		},
		{
			name: "invalid uuid",
			mutate: func(profile *UserProfile) {
				profile.UUID = "not-a-uuid"
			},
			exErr: ErrMalformedProfile,
		},
		{
			name: "invalid ui theme",
			mutate: func(profile *UserProfile) {
				profile.UIPreferences.Theme = ThemeMode("invalid")
			},
			exErr: ErrMalformedProfile,
		},
		{
			name: "nil subscriptions maps",
			mutate: func(profile *UserProfile) {
				profile.Subscriptions.Maps = nil
			},
			exErr: ErrMalformedProfile,
		},
		{
			name: "nil favorites authors",
			mutate: func(profile *UserProfile) {
				profile.Favorites.Authors = nil
			},
			exErr: ErrMalformedProfile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInvalidProfile(t, tt.mutate, tt.exErr)
		})
	}
}

func TestIsValidSubscriptionAction(t *testing.T) {
	require.True(t, IsValidSubscriptionAction(SubscriptionActionSubscribe))
	require.True(t, IsValidSubscriptionAction(SubscriptionActionUnsubscribe))
	require.False(t, IsValidSubscriptionAction(SubscriptionAction("invalid")))
}
