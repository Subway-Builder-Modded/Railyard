package profiles

import (
	"strings"
	"testing"

	"railyard/internal/config"
	"railyard/internal/registry"
	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestUpdateSubscriptionsSubscribeMapAddsOperationRuntimeOnly(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
	}

	result := svc.UpdateSubscriptions(req)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.False(t, result.Persisted)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.Maps["map-a"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, "map-a", result.Operations[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Operations[0].Type)
	require.Equal(t, types.SubscriptionActionSubscribe, result.Operations[0].Action)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Empty(t, persisted.Profiles[types.DefaultProfileID].Subscriptions.Maps)
}

func TestUpdateSubscriptionsForceSyncPersistsStateAndSyncs(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Mods["mod-a"] = "2.0.0"
	state.Profiles[types.DefaultProfileID] = profile

	svc, cfg, reg := loadedUserProfilesServiceWithDependencies(t, state)
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	reg.AddInstalledMod("mod-a", "2.0.0", false)
	materializeInstalledAssets(t, cfg, []types.InstalledModInfo{{ID: "mod-a", Version: "2.0.0"}}, nil)

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod},
		},
		ApplyMode: types.UpdateSubscriptionsPersistAndSync,
	}

	result := svc.UpdateSubscriptions(req)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.True(t, result.Persisted)
	_, exists := result.Profile.Subscriptions.Mods["mod-a"]
	require.False(t, exists)
	require.Len(t, result.Operations, 1)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	_, exists = persisted.Profiles[types.DefaultProfileID].Subscriptions.Mods["mod-a"]
	require.False(t, exists)
	require.Empty(t, reg.GetInstalledMods())
}

func TestUpdateSubscriptionsRepeatedSubscribeSameVersionEmitsOperation(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Maps["map-a"] = "1.2.3"
	state.Profiles[types.DefaultProfileID] = profile
	svc := loadedUserProfilesService(t, state)

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Len(t, result.Operations, 1)
	require.Equal(t, "map-a", result.Operations[0].AssetID)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)
}

func TestUpdateSubscriptionsUnsubscribeRemovesAndEmitsOperation(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Mods["mod-a"] = "3.1.0"
	state.Profiles[types.DefaultProfileID] = profile
	svc := loadedUserProfilesService(t, state)

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod},
		},
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Len(t, result.Operations, 1)
	require.Equal(t, types.Version("3.1.0"), result.Operations[0].Version)
	_, exists := result.Profile.Subscriptions.Mods["mod-a"]
	require.False(t, exists)
}

func TestUpdateSubscriptionsUnsubscribeMissingEntryIsNoOp(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"missing": {Type: types.AssetTypeMap},
		},
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Empty(t, result.Operations)
}

func TestUpdateSubscriptionsUnsubscribeLocalMap(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.LocalMaps["AAA"] = "1.0.0"
	state.Profiles[types.DefaultProfileID] = profile
	svc := loadedUserProfilesService(t, state)

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"AAA": {Type: types.AssetTypeMap},
		},
	})

	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Len(t, result.Operations, 1)
	require.Equal(t, "AAA", result.Operations[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Operations[0].Type)
	require.Equal(t, types.SubscriptionActionUnsubscribe, result.Operations[0].Action)
	require.Equal(t, types.Version("1.0.0"), result.Operations[0].Version)
	_, exists := result.Profile.Subscriptions.LocalMaps["AAA"]
	require.False(t, exists)
}

func TestUpdateSubscriptionsSubscribeLocalMap(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"AAA": {
				Type:    types.AssetTypeMap,
				Version: types.Version("1.2.3"),
				IsLocal: true,
			},
		},
	})

	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Len(t, result.Operations, 1)
	require.Equal(t, "AAA", result.Operations[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Operations[0].Type)
	require.Equal(t, types.SubscriptionActionSubscribe, result.Operations[0].Action)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.LocalMaps["AAA"])
	_, exists := result.Profile.Subscriptions.Maps["AAA"]
	require.False(t, exists)
}

func TestUpdateSubscriptionsRejectsInvalidRequests(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: "missing",
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
	})
	requireProfileErrorType(t, result.Errors, types.ErrorProfileNotFound)
	require.Equal(t, types.ResponseError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Equal(t, types.ErrorProfileNotFound, result.Errors[0].ErrorType)
	require.Equal(t, "missing", result.Errors[0].ProfileID)

	result = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionAction("bad-action"),
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetTypeMap, Version: types.Version("1.0.0")},
		},
	})
	requireProfileErrorType(t, result.Errors, types.ErrorInvalidAction)
	require.Equal(t, types.ResponseError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Equal(t, types.ErrorInvalidAction, result.Errors[0].ErrorType)
	require.Equal(t, "asset", result.Errors[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Errors[0].AssetType)

	result = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetType("bad-type"), Version: types.Version("1.0.0")},
		},
	})
	requireProfileErrorType(t, result.Errors, types.ErrorInvalidAssetType)
	require.Equal(t, types.ResponseError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Equal(t, types.ErrorInvalidAssetType, result.Errors[0].ErrorType)
	require.Equal(t, "asset", result.Errors[0].AssetID)
	require.Equal(t, types.AssetType("bad-type"), result.Errors[0].AssetType)

	result = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetTypeMap, Version: types.Version("not-semver")},
		},
	})
	requireProfileErrorType(t, result.Errors, types.ErrorInvalidVersion)
	require.Equal(t, types.ResponseError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Equal(t, types.ErrorInvalidVersion, result.Errors[0].ErrorType)
	require.Equal(t, "asset", result.Errors[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Errors[0].AssetType)

	require.Panics(t, func() {
		_ = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
			ProfileID: types.DefaultProfileID,
			Action:    types.SubscriptionActionSubscribe,
			ApplyMode: types.UpdateSubscriptionsApplyMode("bad-mode"),
			Assets: map[string]types.SubscriptionUpdateItem{
				"asset": {Type: types.AssetTypeMap, Version: types.Version("1.0.0")},
			},
		})
	})
}

func TestUpdateSubscriptionsAcceptsSemverVersionString(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		ApplyMode: types.UpdateSubscriptionsRuntimeOnly,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-x": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.Maps["map-x"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)
}

func TestUpdateSubscriptionsForceSyncErrors(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Maps["map-a"] = "1.0.0"
	state.Profiles[types.DefaultProfileID] = profile

	svc, _, _ := loadedUserProfilesServiceWithDependencies(t, state)

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.1.0")},
		},
		ApplyMode: types.UpdateSubscriptionsPersistAndSync,
	})

	require.Equal(t, types.ResponseError, result.Status)
	require.Equal(t, "Failed to sync subscriptions", result.Message)
	require.NotEmpty(t, result.Errors)
	require.Equal(t, types.ErrorLookupFailed, result.Errors[len(result.Errors)-1].ErrorType)
	require.Equal(t, types.DefaultProfileID, result.Errors[len(result.Errors)-1].ProfileID)
}

func TestUpdateSubscriptionsMapConflictWarnsThenReplaces(t *testing.T) {
	testutil.NewHarness(t)

	localAssetID := "AAA"
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.LocalMaps[localAssetID] = "1.0.0"
	state.Profiles[types.DefaultProfileID] = profile

	svc, cfg, reg := loadedUserProfilesServiceWithDependencies(t, state)
	configureConfig(t, cfg)

	localInstalled := types.InstalledMapInfo{
		ID:      localAssetID,
		Version: "1.0.0",
		IsLocal: true,
		MapConfig: types.ConfigData{
			Code:    "AAA",
			Version: "1.0.0",
			Name:    "Local AAA",
		},
	}
	reg.AddInstalledMap(localInstalled.ID, localInstalled.Version, true, localInstalled.MapConfig)
	materializeInstalledAssets(t, cfg, nil, []types.InstalledMapInfo{localInstalled})

	cleanup := mockRegistry(t, reg, []registryFixture{
		{
			assetID:   "map-b",
			assetType: types.AssetTypeMap,
			versions:  []string{"1.0.0"},
			mapCode:   "AAA",
		},
	})
	defer cleanup()

	warnResult := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-b": {
				Type:    types.AssetTypeMap,
				Version: types.Version("1.0.0"),
			},
		},
		ApplyMode:         types.UpdateSubscriptionsPersistAndSync,
		ReplaceOnConflict: false,
	})
	require.Equal(t, types.ResponseWarn, warnResult.Status)
	require.NotEmpty(t, warnResult.Conflicts)
	require.Equal(t, localAssetID, warnResult.Conflicts[0].ExistingAssetID)
	require.Contains(t, warnResult.Profile.Subscriptions.LocalMaps, localAssetID)
	require.NotContains(t, warnResult.Profile.Subscriptions.Maps, "map-b")

	replaceResult := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-b": {
				Type:    types.AssetTypeMap,
				Version: types.Version("1.0.0"),
			},
		},
		ApplyMode:         types.UpdateSubscriptionsPersistAndSync,
		ReplaceOnConflict: true,
	})
	require.NotEqual(t, types.ResponseError, replaceResult.Status, replaceResult.Message)
	require.Equal(t, "1.0.0", replaceResult.Profile.Subscriptions.Maps["map-b"])
	_, exists := replaceResult.Profile.Subscriptions.LocalMaps[localAssetID]
	require.False(t, exists)
}

func TestUpdateSubscriptionsToLatest(t *testing.T) {
	type expectation struct {
		expectedStatus          types.Status
		expectedRequestType     types.UpdateSubscriptionRequestType
		expectedHasUpdates      bool
		expectedPendingCount    int
		expectedPendingByKey    map[string][2]string
		expectedApplied         bool
		expectedPersisted       bool
		expectedOperationByID   map[string]string
		expectedMapSubscription string
		expectedModID           string
		expectedModSubscription string
		expectedWarnContains    string
		expectedErrContains     string
	}

	testCases := []struct {
		name         string
		profileID    string
		apply        bool
		targets      []types.SubscriptionUpdateTarget
		state        types.UserProfilesState
		setup        func(t *testing.T, cfg *config.Config, reg *registry.Registry) func()
		expected     expectation
		assertResult func(t *testing.T, svc *UserProfiles, reg *registry.Registry, result types.UpdateSubscriptionsResult)
	}{
		{
			name:      "Updates map and mod to latest semver and syncs",
			profileID: types.DefaultProfileID,
			apply:     true,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.2.0", "2.0.0"}, // newer version(s) available
						mapCode:   "AAA",
					},
					{
						assetID:   "mod-a",
						assetType: types.AssetTypeMod,
						versions:  []string{"1.0.0", "1.5.0"}, // single new version available
					},
				})
			},
			expected: expectation{
				expectedStatus:       types.ResponseSuccess,
				expectedRequestType:  types.LatestApply,
				expectedHasUpdates:   true,
				expectedPendingCount: 2,
				expectedPendingByKey: map[string][2]string{
					"map:map-a": {"1.0.0", "2.0.0"},
					"mod:mod-a": {"1.0.0", "1.5.0"},
				},
				expectedApplied:         true,
				expectedPersisted:       true,
				expectedOperationByID:   map[string]string{"map-a": "2.0.0", "mod-a": "1.5.0"},
				expectedMapSubscription: "2.0.0", // middle version is skipped
				expectedModID:           "mod-a",
				expectedModSubscription: "1.5.0",
			},
			assertResult: func(t *testing.T, _ *UserProfiles, reg *registry.Registry, _ types.UpdateSubscriptionsResult) {
				t.Helper()
				require.Len(t, reg.GetInstalledMaps(), 1)
				require.Equal(t, "map-a", reg.GetInstalledMaps()[0].ID)
				require.Equal(t, "2.0.0", reg.GetInstalledMaps()[0].Version)
				require.Len(t, reg.GetInstalledMods(), 1)
				require.Equal(t, "mod-a", reg.GetInstalledMods()[0].ID)
				require.Equal(t, "1.5.0", reg.GetInstalledMods()[0].Version)
			},
		},
		{
			name:      "No-op when all subscriptions are up-to-date",
			profileID: types.DefaultProfileID,
			apply:     true,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "2.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.5.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, _ *config.Config, reg *registry.Registry) func() {
				t.Helper()
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "2.0.0"}, // already latest version
						mapCode:   "AAA",
					},
					{
						assetID:   "mod-a",
						assetType: types.AssetTypeMod,
						versions:  []string{"1.0.0", "1.5.0"}, // already latest version
					},
				})
			},
			expected: expectation{
				expectedStatus:          types.ResponseSuccess,
				expectedRequestType:     types.LatestApply,
				expectedHasUpdates:      false,
				expectedPendingCount:    0,
				expectedPendingByKey:    map[string][2]string{},
				expectedApplied:         false,
				expectedPersisted:       false,
				expectedOperationByID:   map[string]string{},
				expectedMapSubscription: "2.0.0",
				expectedModID:           "mod-a",
				expectedModSubscription: "1.5.0",
			},
		},
		{
			name:      "Lookup failures warn but do not prevent request completion",
			profileID: types.DefaultProfileID,
			apply:     true,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-missing"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				reg.AddInstalledMod("mod-missing", "1.0.0", false)
				// Previously installed mod is now missing from registry (causing a lookup warning)
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.1.0"},
						mapCode:   "AAA",
					},
				})
			},
			expected: expectation{
				expectedStatus:       types.ResponseWarn,
				expectedRequestType:  types.LatestApply,
				expectedHasUpdates:   true,
				expectedPendingCount: 1,
				expectedPendingByKey: map[string][2]string{
					"map:map-a": {"1.0.0", "1.1.0"},
				},
				expectedApplied:         true,
				expectedPersisted:       true, // state is updated for map-a but not mod-missing
				expectedOperationByID:   map[string]string{"map-a": "1.1.0"},
				expectedMapSubscription: "1.1.0",
				expectedModID:           "mod-missing",
				expectedModSubscription: "1.0.0",
				expectedWarnContains:    "Updated 1 subscriptions; skipped 1 subscriptions",
			},
		},
		{
			name:      "All lookups fail returns warning and no operations but requests completes",
			profileID: types.DefaultProfileID,
			apply:     true,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, _ *config.Config, reg *registry.Registry) func() {
				t.Helper()
				return mockRegistry(t, reg, nil) // Neither installed map nor installed mod are present
			},
			expected: expectation{
				expectedStatus:          types.ResponseWarn,
				expectedRequestType:     types.LatestApply,
				expectedHasUpdates:      false,
				expectedPendingCount:    0,
				expectedPendingByKey:    map[string][2]string{},
				expectedApplied:         false,
				expectedPersisted:       false, // no state updates occur
				expectedOperationByID:   map[string]string{},
				expectedMapSubscription: "1.0.0",
				expectedModID:           "mod-a",
				expectedModSubscription: "1.0.0",
				expectedWarnContains:    "no updates applied; skipped 2 subscriptions",
			},
		},
		{
			name:      "Sync failure is propagated as error",
			profileID: types.DefaultProfileID,
			apply:     true,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, _ *config.Config, reg *registry.Registry) func() {
				t.Helper()
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.1.0"},
						mapCode:   "AAA",
					},
				})
			},
			expected: expectation{
				expectedStatus:       types.ResponseError,
				expectedRequestType:  types.LatestApply,
				expectedHasUpdates:   true,
				expectedPendingCount: 1,
				expectedPendingByKey: map[string][2]string{
					"map:map-a": {"1.0.0", "1.1.0"},
				},
				expectedApplied:         true,
				expectedPersisted:       true, // state is updated to desired but sync fails
				expectedOperationByID:   map[string]string{"map-a": "1.1.0"},
				expectedMapSubscription: "1.1.0",
				expectedErrContains:     `Failed sync action: subscribe map "map-a" failed`,
			},
		},
		{
			name:      "Check mode reports pending updates without applying",
			profileID: types.DefaultProfileID,
			apply:     false,
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, _ *config.Config, reg *registry.Registry) func() {
				t.Helper()
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.2.0"},
						mapCode:   "AAA",
					},
					{
						assetID:   "mod-a",
						assetType: types.AssetTypeMod,
						versions:  []string{"1.0.0", "1.5.0"},
					},
				})
			},
			expected: expectation{
				expectedStatus:       types.ResponseSuccess,
				expectedRequestType:  types.LatestCheck,
				expectedHasUpdates:   true,
				expectedPendingCount: 2,
				expectedPendingByKey: map[string][2]string{
					"map:map-a": {"1.0.0", "1.2.0"},
					"mod:mod-a": {"1.0.0", "1.5.0"},
				},
				expectedApplied:         false,
				expectedPersisted:       false,
				expectedOperationByID:   map[string]string{},
				expectedMapSubscription: "1.0.0",
				expectedModID:           "mod-a",
				expectedModSubscription: "1.0.0",
			},
		},
		{
			name:      "Check mode with targets only reports targeted pending updates",
			profileID: types.DefaultProfileID,
			apply:     false,
			targets: []types.SubscriptionUpdateTarget{
				{AssetID: "mod-a", Type: types.AssetTypeMod},
			},
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, _ *config.Config, reg *registry.Registry) func() {
				t.Helper()
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.2.0"},
						mapCode:   "AAA",
					},
					{
						assetID:   "mod-a",
						assetType: types.AssetTypeMod,
						versions:  []string{"1.0.0", "1.5.0"},
					},
				})
			},
			expected: expectation{
				expectedStatus:          types.ResponseSuccess,
				expectedRequestType:     types.LatestCheck,
				expectedHasUpdates:      true,
				expectedPendingCount:    1,
				expectedPendingByKey:    map[string][2]string{"mod:mod-a": {"1.0.0", "1.5.0"}},
				expectedApplied:         false,
				expectedPersisted:       false,
				expectedOperationByID:   map[string]string{},
				expectedMapSubscription: "1.0.0",
				expectedModID:           "mod-a",
				expectedModSubscription: "1.0.0",
			},
		},
		{
			name:      "Apply mode with targets updates only targeted subscriptions",
			profileID: types.DefaultProfileID,
			apply:     true,
			targets: []types.SubscriptionUpdateTarget{
				{AssetID: "map-a", Type: types.AssetTypeMap},
			},
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.0"
				profile.Subscriptions.Mods["mod-a"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			setup: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:   "map-a",
						assetType: types.AssetTypeMap,
						versions:  []string{"1.0.0", "1.2.0"},
						mapCode:   "AAA",
					},
					{
						assetID:   "mod-a",
						assetType: types.AssetTypeMod,
						versions:  []string{"1.0.0", "1.5.0"},
					},
				})
			},
			expected: expectation{
				expectedStatus:       types.ResponseSuccess,
				expectedRequestType:  types.LatestApply,
				expectedHasUpdates:   true,
				expectedPendingCount: 1,
				expectedPendingByKey: map[string][2]string{
					"map:map-a": {"1.0.0", "1.2.0"},
				},
				expectedApplied:         true,
				expectedPersisted:       true,
				expectedOperationByID:   map[string]string{"map-a": "1.2.0"},
				expectedMapSubscription: "1.2.0",
				expectedModID:           "mod-a",
				expectedModSubscription: "1.0.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.NewHarness(t)
			svc, cfg, reg := loadedUserProfilesServiceWithDependencies(t, tc.state)
			var cleanup func()
			if tc.setup != nil {
				cleanup = tc.setup(t, cfg, reg)
			}
			if cleanup != nil {
				defer cleanup()
			}

			result := svc.UpdateSubscriptionsToLatest(types.UpdateSubscriptionsToLatestRequest{
				ProfileID: tc.profileID,
				Apply:     tc.apply,
				Targets:   tc.targets,
			})
			require.Equal(t, tc.expected.expectedStatus, result.Status)
			require.Equal(t, tc.expected.expectedRequestType, result.RequestType)
			require.Equal(t, tc.expected.expectedHasUpdates, result.HasUpdates)
			require.Equal(t, tc.expected.expectedPendingCount, result.PendingCount)
			pendingByKey := map[string][2]string{}
			for _, pending := range result.PendingUpdates {
				key := string(pending.Type) + ":" + pending.AssetID
				pendingByKey[key] = [2]string{
					strings.TrimSpace(string(pending.CurrentVersion)),
					strings.TrimSpace(string(pending.LatestVersion)),
				}
			}
			require.Equal(t, tc.expected.expectedPendingByKey, pendingByKey)
			require.Equal(t, tc.expected.expectedApplied, result.Applied)
			if tc.expected.expectedErrContains != "" {
				require.NotEmpty(t, result.Errors)
				found := false
				for _, profileErr := range result.Errors {
					if strings.Contains(profileErr.Error(), tc.expected.expectedErrContains) {
						found = true
						break
					}
				}
				require.True(t, found)
			}

			require.Equal(t, tc.expected.expectedPersisted, result.Persisted)
			if tc.expected.expectedWarnContains != "" {
				require.Contains(t, result.Message, tc.expected.expectedWarnContains)
				require.NotEmpty(t, result.Errors)
			}

			operationByID := map[string]string{}
			for _, operation := range result.Operations {
				operationByID[operation.AssetID] = strings.TrimSpace(string(operation.Version))
			}
			require.Equal(t, tc.expected.expectedOperationByID, operationByID)

			if tc.expected.expectedMapSubscription != "" {
				require.Equal(t, tc.expected.expectedMapSubscription, result.Profile.Subscriptions.Maps["map-a"])
			}
			if tc.expected.expectedModSubscription != "" && tc.expected.expectedModID != "" {
				require.Equal(t, tc.expected.expectedModSubscription, result.Profile.Subscriptions.Mods[tc.expected.expectedModID])
			}

			if tc.assertResult != nil {
				tc.assertResult(t, svc, reg, result)
			}

			persisted, readErr := ReadUserProfilesState()
			require.NoError(t, readErr)
			persistedProfile := persisted.Profiles[types.DefaultProfileID]
			if tc.expected.expectedMapSubscription != "" {
				require.Equal(t, tc.expected.expectedMapSubscription, persistedProfile.Subscriptions.Maps["map-a"])
			}
			if tc.expected.expectedModSubscription != "" && tc.expected.expectedModID != "" {
				require.Equal(t, tc.expected.expectedModSubscription, persistedProfile.Subscriptions.Mods[tc.expected.expectedModID])
			}
		})
	}
}
