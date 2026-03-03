package main

import (
	"os"
	"path/filepath"
	"railyard/internal/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testUserProfilesLogger(t *testing.T) Logger {
	t.Helper()
	return loggerAtPath(filepath.Join(t.TempDir(), "user_profiles_test.log"))
}

func newUserProfilesService(t *testing.T) *UserProfiles {
	t.Helper()
	return NewUserProfiles(testUserProfilesLogger(t))
}

func newLoadedUserProfilesService(t *testing.T, state types.UserProfilesState) *UserProfiles {
	t.Helper()
	require.NoError(t, writeUserProfilesState(state))

	svc := newUserProfilesService(t)
	_, err := svc.loadProfiles()
	require.NoError(t, err)
	return svc
}

func writeRawUserProfilesFile(t *testing.T, content string) {
	t.Helper()

	path := UserProfilesPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestLoadProfilesBootstrapsAndPersistsStateWhenMissing(t *testing.T) {
	setEnv(t)

	svc := newUserProfilesService(t)
	active, err := svc.loadProfiles()
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

	svc := newUserProfilesService(t)
	_, err := svc.GetActiveProfile()
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

	svc := newUserProfilesService(t)
	_, err := svc.loadProfiles()
	require.ErrorIs(t, err, types.ErrInvalidState)
}

func TestResolveActiveProfileReturnsLoadedActiveProfile(t *testing.T) {
	setEnv(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("custom", "Custom")
	state.ActiveProfileID = custom.ID
	state.Profiles[custom.ID] = custom
	require.NoError(t, writeUserProfilesState(state))

	svc := newUserProfilesService(t)
	loadedActive, err := svc.loadProfiles()
	require.NoError(t, err)
	require.Equal(t, custom.ID, loadedActive.ID)
	require.Equal(t, custom.Name, loadedActive.Name)

	active, err := svc.GetActiveProfile()
	require.NoError(t, err)
	require.Equal(t, custom.ID, active.ID)
	require.Equal(t, custom.Name, active.Name)
}

func TestQuarantineUserProfilesFileMovesSourceToBackup(t *testing.T) {
	setEnv(t)
	writeRawUserProfilesFile(t, "{}")

	svc := newUserProfilesService(t)
	success, backupPath := svc.quarantineUserProfiles()
	require.True(t, success)
	require.NotEmpty(t, backupPath)
	require.True(t, strings.Contains(filepath.Base(backupPath), "user_profiles.invalid."))

	_, err := os.Stat(backupPath)
	require.NoError(t, err)

	_, err = os.Stat(UserProfilesPath())
	require.True(t, os.IsNotExist(err))
}

func TestUpdateSubscriptionsSubscribeMapAddsOperationAndRuntimeOnlyByDefault(t *testing.T) {
	setEnv(t)
	svc := newLoadedUserProfilesService(t, types.InitialProfilesState())

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
		ForceSync: false,
	}

	result, err := svc.UpdateSubscriptions(req)
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.False(t, result.Persisted)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.Maps["map-a"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, "map-a", result.Operations[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Operations[0].Type)
	require.Equal(t, types.SubscriptionActionSubscribe, result.Operations[0].Action)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)

	persisted, err := readUserProfilesState()
	require.NoError(t, err)
	require.Empty(t, persisted.Profiles[types.DefaultProfileID].Subscriptions.Maps)
}

func TestUpdateSubscriptionsSubscribeWithForceSyncPersistsState(t *testing.T) {
	setEnv(t)
	svc := newLoadedUserProfilesService(t, types.InitialProfilesState())

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod, Version: types.Version("2.0.0")},
		},
		ForceSync: true,
	}

	result, err := svc.UpdateSubscriptions(req)
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.True(t, result.Persisted)
	require.Equal(t, "2.0.0", result.Profile.Subscriptions.Mods["mod-a"])
	require.Len(t, result.Operations, 1)

	persisted, err := readUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, "2.0.0", persisted.Profiles[types.DefaultProfileID].Subscriptions.Mods["mod-a"])
}

func TestUpdateSubscriptionsRepeatedSubscribeSameVersionEmitsOperation(t *testing.T) {
	setEnv(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Maps["map-a"] = "1.2.3"
	state.Profiles[types.DefaultProfileID] = profile
	svc := newLoadedUserProfilesService(t, state)

	result, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Len(t, result.Operations, 1)
	require.Equal(t, "map-a", result.Operations[0].AssetID)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)
}

func TestUpdateSubscriptionsUnsubscribeRemovesAndEmitsOperation(t *testing.T) {
	setEnv(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Mods["mod-a"] = "3.1.0"
	state.Profiles[types.DefaultProfileID] = profile
	svc := newLoadedUserProfilesService(t, state)

	result, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Len(t, result.Operations, 1)
	require.Equal(t, types.Version("3.1.0"), result.Operations[0].Version)
	_, exists := result.Profile.Subscriptions.Mods["mod-a"]
	require.False(t, exists)
}

func TestUpdateSubscriptionsUnsubscribeMissingEntryIsNoOp(t *testing.T) {
	setEnv(t)
	svc := newLoadedUserProfilesService(t, types.InitialProfilesState())

	result, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"missing": {Type: types.AssetTypeMap},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Empty(t, result.Operations)
}

func TestUpdateSubscriptionsRejectsInvalidRequests(t *testing.T) {
	setEnv(t)
	svc := newLoadedUserProfilesService(t, types.InitialProfilesState())

	_, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: "missing",
		Action:    types.SubscriptionActionSubscribe,
	})
	require.ErrorIs(t, err, ErrProfileNotFound)

	_, err = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionAction("bad-action"),
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetTypeMap, Version: types.Version("1.0.0")},
		},
	})
	require.ErrorIs(t, err, ErrInvalidSubscriptionAction)

	_, err = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetType("bad-type"), Version: types.Version("1.0.0")},
		},
	})
	require.ErrorIs(t, err, ErrInvalidAssetType)
}

func TestUpdateSubscriptionsAcceptsOpaqueVersionString(t *testing.T) {
	setEnv(t)
	svc := newLoadedUserProfilesService(t, types.InitialProfilesState())

	result, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-x": {Type: types.AssetTypeMap, Version: types.Version("not-semver")},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Equal(t, "not-semver", result.Profile.Subscriptions.Maps["map-x"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, types.Version("not-semver"), result.Operations[0].Version)
}

func newTestUserProfile(id string, name string) types.UserProfile {
	profile := types.DefaultProfile()
	profile.ID = id
	profile.Name = name
	return profile
}
