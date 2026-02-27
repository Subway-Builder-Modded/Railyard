package main

import (
	"testing"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestResolveStartupProfileFallsBackToDefaultWhenProfilesFailToLoad(t *testing.T) {
	setEnv(t)
	writeRawUserProfilesFile(t, "{")

	app := &App{
		Profiles: NewUserProfiles(),
	}

	profile := resolveStartupProfile(app)
	require.Equal(t, types.DefaultProfileID, profile.ID)
	require.Equal(t, types.DefaultProfileName, profile.Name)
	require.True(t, profile.SystemPreferences.RefreshRegistryOnStartup)
	require.False(t, profile.SystemPreferences.AutoUpdateSubscriptions)
}

func TestResolveStartupProfileReturnsLoadedActiveProfile(t *testing.T) {
	setEnv(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("custom", "Custom")
	state.ActiveProfileID = custom.ID
	state.Profiles[custom.ID] = custom
	require.NoError(t, writeUserProfilesState(state))

	app := &App{
		Profiles: NewUserProfiles(),
	}

	profile := resolveStartupProfile(app)
	require.Equal(t, custom.ID, profile.ID)
	require.Equal(t, custom.Name, profile.Name)
}
