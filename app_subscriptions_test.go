package main

import (
	"railyard/internal/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func newLoadedTestApp(t *testing.T) *App {
	t.Helper()
	setEnv(t)

	app := NewApp()
	require.NoError(t, writeUserProfilesState(types.InitialProfilesState()))
	_, err := app.Profiles.loadProfiles()
	require.NoError(t, err)
	return app
}

func TestAppUpdateSubscriptionsForceSyncReturnsOperations(t *testing.T) {
	app := newLoadedTestApp(t)

	result, err := app.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.0.0")},
		},
		ForceSync: true,
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Len(t, result.Operations, 1)
	require.True(t, result.Persisted)
	require.Equal(t, types.DefaultProfileID, result.Profile.ID)
}

func TestAppUpdateSubscriptionsWithoutForceSyncReturnsOperations(t *testing.T) {
	app := newLoadedTestApp(t)

	result, err := app.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod, Version: types.Version("1.0.0")},
		},
		ForceSync: false,
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Len(t, result.Operations, 1)
	require.False(t, result.Persisted)
}

func TestAppUpdateSubscriptionsBubblesProfileError(t *testing.T) {
	app := newLoadedTestApp(t)

	_, err := app.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: "missing-profile",
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("2.0.0")},
		},
		ForceSync: true,
	})
	require.ErrorIs(t, err, ErrProfileNotFound)
}
