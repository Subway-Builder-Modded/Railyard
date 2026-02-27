package main

import (
	"os"
	"path/filepath"
	"railyard/internal/types"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func writeRawUserProfilesFile(t *testing.T, content string) {
	t.Helper()

	path := UserProfilesPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestLoadProfilesBootstrapsAndPersistsStateWhenMissing(t *testing.T) {
	setEnv(t)

	svc := NewUserProfiles(NewMockLogger())
	active, err := svc.LoadProfiles()
	require.NoError(t, err)
	require.Equal(t, types.DefaultProfileID, active.ID)
	require.Equal(t, types.DefaultProfileName, active.Name)

	persisted, err := readUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, types.DefaultProfileID, persisted.ActiveProfileID)

	defaultProfile, ok := persisted.Profiles[types.DefaultProfileID]
	require.True(t, ok)
	require.Equal(t, types.DefaultProfileID, defaultProfile.ID)
	require.Equal(t, types.DefaultProfileName, defaultProfile.Name)
	require.NotEmpty(t, defaultProfile.UUID)
}

func TestResolveActiveProfileFailsWhenNotLoaded(t *testing.T) {
	setEnv(t)

	svc := NewUserProfiles(NewMockLogger())
	_, err := svc.ResolveActiveProfile()
	require.ErrorIs(t, err, ErrProfilesNotLoaded)
}

func TestLoadProfilesReturnsErrorForInvalidState(t *testing.T) {
	setEnv(t)

	invalid := types.UserProfilesState{
		ActiveProfileID: "custom",
		Profiles: map[string]types.UserProfile{
			"custom": newTestUserProfile("custom", "Custom"),
		},
	}
	require.NoError(t, writeUserProfilesState(invalid))

	svc := NewUserProfiles(NewMockLogger())
	_, err := svc.LoadProfiles()
	require.ErrorIs(t, err, types.ErrInvalidState)
}

func TestResolveActiveProfileReturnsLoadedActiveProfile(t *testing.T) {
	setEnv(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("custom", "Custom")
	state.ActiveProfileID = custom.ID
	state.Profiles[custom.ID] = custom
	require.NoError(t, writeUserProfilesState(state))

	svc := NewUserProfiles(NewMockLogger())
	loadedActive, err := svc.LoadProfiles()
	require.NoError(t, err)
	require.Equal(t, custom.ID, loadedActive.ID)
	require.Equal(t, custom.Name, loadedActive.Name)

	active, err := svc.ResolveActiveProfile()
	require.NoError(t, err)
	require.Equal(t, custom.ID, active.ID)
	require.Equal(t, custom.Name, active.Name)
}

func TestQuarantineUserProfilesFileMovesSourceToBackup(t *testing.T) {
	setEnv(t)
	writeRawUserProfilesFile(t, "{}")

	svc := NewUserProfiles(NewMockLogger())
	success, backupPath := svc.quarantineUserProfiles()
	require.True(t, success)
	require.NotEmpty(t, backupPath)
	require.True(t, strings.Contains(filepath.Base(backupPath), "user_profiles.invalid."))

	_, err := os.Stat(backupPath)
	require.NoError(t, err)

	_, err = os.Stat(UserProfilesPath())
	require.True(t, os.IsNotExist(err))
}

func newTestUserProfile(id string, name string) types.UserProfile {
	return types.UserProfile{
		ID:   id,
		UUID: uuid.NewString(),
		Name: name,
		UIPreferences: types.UIPreferences{
			Theme:          types.ThemeDark,
			DefaultPerPage: types.PageSize12,
		},
		SystemPreferences: types.SystemPreferences{
			RefreshRegistryOnStartup: true,
			AutoUpdateSubscriptions:  false,
		},
		Subscriptions: types.Subscriptions{
			Maps: map[string]string{},
			Mods: map[string]string{},
		},
		Favorites: types.Favorites{
			Authors: []string{},
			Maps:    []string{},
			Mods:    []string{},
		},
	}
}
