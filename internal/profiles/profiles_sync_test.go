package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"railyard/internal/config"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestSyncActionErrorIgnoresWarnings(t *testing.T) {
	t.Run("Duplicate install warning returns no error", func(t *testing.T) {
		err := syncInstallActionError(
			types.SubscriptionActionSubscribe,
			types.AssetTypeMap,
			"map-a",
			types.AssetInstallResponse{
				GenericResponse: types.GenericResponse{Status: types.ResponseWarn, Message: "Duplicate request skipped: install already queued"},
				AssetType:       types.AssetTypeMap,
				AssetID:         "map-a",
			},
		)
		require.NoError(t, err)
	})

	t.Run("Duplicate uninstall warning returns no error", func(t *testing.T) {
		err := syncUninstallActionError(
			types.SubscriptionActionUnsubscribe,
			types.AssetTypeMod,
			"mod-a",
			types.AssetUninstallResponse{
				GenericResponse: types.GenericResponse{Status: types.ResponseWarn, Message: "Duplicate request skipped: uninstall already queued"},
				AssetType:       types.AssetTypeMod,
				AssetID:         "mod-a",
			},
		)
		require.NoError(t, err)
	})

	t.Run("Non-duplicate warning still returns error", func(t *testing.T) {
		err := syncInstallActionError(
			types.SubscriptionActionSubscribe,
			types.AssetTypeMap,
			"map-a",
			types.AssetInstallResponse{
				GenericResponse: types.GenericResponse{Status: types.ResponseWarn, Message: "Map with ID map-a is not currently installed. No action taken."},
				AssetType:       types.AssetTypeMap,
				AssetID:         "map-a",
			},
		)
		require.NoError(t, err)
	})
}

func TestSyncSubscriptions(t *testing.T) {
	type expectedState struct {
		mods []types.InstalledModInfo
		maps []types.InstalledMapInfo
	}

	testCases := []struct {
		name string

		state       types.UserProfilesState
		initialMods []types.InstalledModInfo
		initialMaps []types.InstalledMapInfo

		prepare             func(t *testing.T, cfg *config.Config, reg *registry.Registry) func()
		assertSubscriptions func(t *testing.T, svc *UserProfiles)

		expectedStatus     types.Status
		expectedErrors     []string
		expectedErrorTypes []types.UserProfilesErrorType
		expectedState      expectedState
	}{
		{
			name:  "No subscriptions with no installed assets is no-op",
			state: types.InitialProfilesState(),
			expectedState: expectedState{
				mods: nil,
				maps: nil,
			},
			assertSubscriptions: func(t *testing.T, svc *UserProfiles) {
				t.Helper()
				active := svc.GetActiveProfile()
				require.Equal(t, types.ResponseSuccess, active.Status)
				_, exists := active.Profile.Subscriptions.Mods["mod-b"]
				require.False(t, exists)

				persisted, err := ReadUserProfilesState()
				require.NoError(t, err)
				_, exists = persisted.Profiles[types.DefaultProfileID].Subscriptions.Mods["mod-b"]
				require.False(t, exists)
			},
		},
		{
			name:  "Unsubscribed installed mod is removed",
			state: types.InitialProfilesState(),
			initialMods: []types.InstalledModInfo{
				{ID: "mod-a", Version: "1.0.0"},
			},
			prepare: func(t *testing.T, cfg *config.Config, _ *registry.Registry) func() {
				t.Helper()
				cfg.Cfg.MetroMakerDataPath = t.TempDir()
				return nil
			},
			expectedState: expectedState{
				mods: []types.InstalledModInfo{},
				maps: nil,
			},
		},
		{
			name:  "Unsubscribed installed map is removed",
			state: types.InitialProfilesState(),
			initialMaps: []types.InstalledMapInfo{
				{
					ID:      "map-a",
					Version: "2.0.0",
					MapConfig: types.ConfigData{
						Code: "AAA",
					},
				},
			},
			prepare: func(t *testing.T, cfg *config.Config, _ *registry.Registry) func() {
				t.Helper()
				cfg.Cfg.MetroMakerDataPath = t.TempDir()
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return nil
			},
			expectedState: expectedState{
				mods: nil,
				maps: []types.InstalledMapInfo{},
			},
		},
		{
			name: "Sync errors when subscribed assets are unavailable",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Mods["mod-b"] = "1.0.0"
				profile.Subscriptions.Maps["map-b"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			// Neither mod-b nor map-b are present
			initialMods: []types.InstalledModInfo{
				{ID: "mod-a", Version: "1.0.0"},
			},
			initialMaps: []types.InstalledMapInfo{
				{
					ID:      "map-a",
					Version: "1.0.0",
					MapConfig: types.ConfigData{
						Code: "AAA",
					},
				},
			},
			prepare: func(t *testing.T, cfg *config.Config, _ *registry.Registry) func() {
				t.Helper()
				cfg.Cfg.MetroMakerDataPath = t.TempDir()
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return nil
			},
			// Error occurs during attempt to install unavailable errors
			expectedErrors: []string{
				`Subscribe mod "mod-b" failed`,
				`Subscribe map "map-b" failed`,
			},
			expectedState: expectedState{
				mods: []types.InstalledModInfo{},
				maps: []types.InstalledMapInfo{},
			},
		},
		{
			name: "Sync succeeds when subscribed assets are available",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Mods["mod-b"] = "1.0.0"
				profile.Subscriptions.Maps["map-b"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			initialMods: []types.InstalledModInfo{
				{ID: "mod-a", Version: "1.0.0"},
			},
			initialMaps: []types.InstalledMapInfo{
				{
					ID:      "map-a",
					Version: "1.0.0",
					MapConfig: types.ConfigData{
						Code: "AAA",
					},
				},
			},
			prepare: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return mockRegistry(t, reg, []registryFixture{
					{assetID: "mod-b", versions: []string{"1.0.0"}, assetType: types.AssetTypeMod},
					{assetID: "map-b", versions: []string{"1.0.0"}, assetType: types.AssetTypeMap, mapCode: "BBB"},
				})
			},
			expectedState: expectedState{
				mods: []types.InstalledModInfo{
					{ID: "mod-b", Version: "1.0.0"},
				},
				maps: []types.InstalledMapInfo{
					{
						ID:      "map-b",
						Version: "1.0.0",
						MapConfig: types.ConfigData{
							Code: "BBB",
							Name: "Fixture Map",
						},
					},
				},
			},
		},
		{
			name: "Sync errors when subscribed mod archive is missing manifest",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Mods["mod-b"] = "1.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			prepare: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				return mockRegistry(t, reg, []registryFixture{
					{
						assetID:            "mod-b",
						assetType:          types.AssetTypeMod,
						versions:           []string{"1.0.0"},
						missingModManifest: true,
					},
				})
			},
			expectedStatus: types.ResponseWarn,
			expectedState: expectedState{
				mods: nil,
				maps: nil,
			},
		},
		{
			name: "Sync errors on attempted update to new version of asset that is not available",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.1"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			initialMaps: []types.InstalledMapInfo{
				{
					ID:      "map-a",
					Version: "1.0.0",
					MapConfig: types.ConfigData{
						Code: "AAA",
					},
				},
			},
			expectedErrors: []string{
				`Subscribe map "map-a" failed`,
			},
			expectedState: expectedState{
				mods: nil,
				maps: []types.InstalledMapInfo{
					{
						ID:      "map-a",
						Version: "1.0.0",
						MapConfig: types.ConfigData{
							Code: "AAA",
						},
					},
				},
			},
		},
		{
			name: "Sync succeeds on attempted update to new version of asset that is available",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.1"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			initialMaps: []types.InstalledMapInfo{
				{
					ID:      "map-a",
					Version: "1.0.0",
					MapConfig: types.ConfigData{
						Code: "AAA",
					},
				},
			},
			prepare: func(t *testing.T, cfg *config.Config, reg *registry.Registry) func() {
				t.Helper()
				configureConfig(t, cfg)
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return mockRegistry(t, reg, []registryFixture{
					{assetID: "map-a", versions: []string{"1.0.1"}, assetType: types.AssetTypeMap, mapCode: "AAA"},
				})
			},
			expectedState: expectedState{
				mods: nil,
				maps: []types.InstalledMapInfo{
					// Previously present version (1.0.0) should now be removed
					{
						ID:      "map-a",
						Version: "1.0.1",
						MapConfig: types.ConfigData{
							Code: "AAA",
							Name: "Fixture Map",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.NewHarness(t)

			svc, cfg, reg := loadedUserProfilesServiceWithDependencies(t, tc.state)
			configureConfig(t, cfg)
			for _, mod := range tc.initialMods {
				reg.AddInstalledMod(mod.ID, mod.Version, false)
			}
			for _, m := range tc.initialMaps {
				reg.AddInstalledMap(m.ID, m.Version, false, m.MapConfig)
			}
			var cleanup func()
			if tc.prepare != nil {
				cleanup = tc.prepare(t, cfg, reg)
			}
			materializeInstalledAssets(t, cfg, tc.initialMods, tc.initialMaps)
			if cleanup != nil {
				defer cleanup()
			}

			result := svc.SyncSubscriptions(types.DefaultProfileID, false)
			expectedStatus := tc.expectedStatus
			if expectedStatus == "" {
				if len(tc.expectedErrors) == 0 {
					expectedStatus = types.ResponseSuccess
				} else {
					expectedStatus = types.ResponseError
				}
			}
			require.Equal(t, expectedStatus, result.Status)

			if len(tc.expectedErrors) > 0 {
				for _, expected := range tc.expectedErrors {
					found := false
					for _, profileErr := range result.Errors {
						if strings.Contains(profileErr.Error(), expected) {
							found = true
							break
						}
					}
					require.Truef(t, found, "expected error substring %q not found in %+v", expected, result.Errors)
				}
				for _, expectedErrorType := range tc.expectedErrorTypes {
					found := false
					for _, profileErr := range result.Errors {
						if profileErr.ErrorType == expectedErrorType {
							found = true
							break
						}
					}
					require.Truef(t, found, "expected error type %q not found in %+v", expectedErrorType, result.Errors)
				}
			}

			require.Equal(t, tc.expectedState.mods, reg.GetInstalledMods())
			require.Equal(t, tc.expectedState.maps, reg.GetInstalledMaps())
			if tc.assertSubscriptions != nil {
				tc.assertSubscriptions(t, svc)
			}
		})
	}
}

func TestSyncAssetSubscriptionsInstallDecisionsMaps(t *testing.T) {
	testCases := []struct {
		name               string
		subscriptions      map[string]string
		installedVersion   map[string]string
		availableVersions  map[string]map[string]struct{}
		expectedInstalls   int
		expectedUninstalls int
		expectedErrors     []string
	}{
		{
			// TODO: Add warning log to implementation and validate warning here
			name: "Already installed version skips install even when unavailable",
			subscriptions: map[string]string{
				"map-a": "1.0.0",
			},
			installedVersion: map[string]string{
				"map-a": "1.0.0",
			},
			availableVersions:  map[string]map[string]struct{}{},
			expectedInstalls:   0,
			expectedUninstalls: 0,
			expectedErrors:     nil,
		},
		// TODO: We should probably raise an error if the installed version is no longer available...
		// But that is an issue with the registry and not the UserProfiles
		{
			name: "Available newer version triggers install and updates index",
			subscriptions: map[string]string{
				"map-a": "1.0.1",
			},
			installedVersion: map[string]string{
				"map-a": "1.0.0",
			},
			availableVersions: map[string]map[string]struct{}{
				"map-a": {
					"1.0.1": {},
					"1.0.0": {},
				},
			},
			expectedInstalls:   1,
			expectedUninstalls: 1,
			expectedErrors:     nil,
		},
		{
			name: "Unavailable version blocks install",
			subscriptions: map[string]string{
				"map-a": "2.0.0",
			},
			installedVersion: map[string]string{
				"map-a": "1.0.0",
			},
			availableVersions: map[string]map[string]struct{}{
				"map-a": {
					"1.0.1": {},
					"1.0.0": {},
				},
			},
			expectedInstalls:   0,
			expectedUninstalls: 0,
			expectedErrors: []string{
				`Subscribe map "map-a" failed: version "2.0.0" is not available`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			installCalls := 0
			uninstallCalls := 0
			_, errs, _, _ := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockMapAssetSyncArgs(assetSyncTestFixture{
				subscriptions:     tc.subscriptions,
				installedVersion:  tc.installedVersion,
				availableVersions: tc.availableVersions,
			},
				mockInstallResponse(types.AssetTypeMap, &installCalls, nil),
				mockUninstallResponse(types.AssetTypeMap, &uninstallCalls, nil),
			))

			require.Equal(t, tc.expectedInstalls, installCalls)
			require.Equal(t, tc.expectedUninstalls, uninstallCalls)
			if len(tc.expectedErrors) == 0 {
				require.Empty(t, errs)
			} else {
				require.Len(t, errs, len(tc.expectedErrors))
				for i, expected := range tc.expectedErrors {
					require.Contains(t, errs[i].Error(), expected)
				}
			}
		})
	}
}

func TestSyncAssetSubscriptionsPropagatesInstallErrors(t *testing.T) {
	fixture := assetSyncTestFixture{
		subscriptions: map[string]string{
			"map-a": "1.0.1",
		},
		installedVersion: map[string]string{
			"map-a": "1.0.0",
		},
		availableVersions: map[string]map[string]struct{}{
			"map-a": {
				"1.0.1": {},
			},
		},
	}

	installCalls := 0
	uninstallCalls := 0
	_, errs, assetsToPurge, _ := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockMapAssetSyncArgs(
		fixture,
		mockInstallResponse(types.AssetTypeMap, &installCalls, map[string]types.AssetInstallResponse{
			"map-a": {
				GenericResponse: types.GenericResponse{
					Status:  types.ResponseError,
					Message: "Failed to extract map zip: Cannot install map because its code matches a vanilla map included with the game or an already installed map.",
				},
				ErrorType: types.InstallErrorExtractFailed,
			},
		}),
		mockUninstallResponse(types.AssetTypeMap, &uninstallCalls, nil),
	))

	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "Failed to extract map zip")
	require.Equal(t, 1, installCalls)
	require.Equal(t, 1, uninstallCalls)
	require.Empty(t, assetsToPurge)
}

func TestSyncAssetSubscriptionsChecksumFailureProducesPurgeCandidate(t *testing.T) {
	fixture := assetSyncTestFixture{
		subscriptions: map[string]string{
			"map-a": "1.0.1",
		},
		installedVersion: map[string]string{
			"map-a": "1.0.0",
		},
		availableVersions: map[string]map[string]struct{}{
			"map-a": {
				"1.0.1": {},
			},
		},
	}

	_, errs, assetsToPurge, _ := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockMapAssetSyncArgs(
		fixture,
		mockInstallResponse(types.AssetTypeMap, nil, map[string]types.AssetInstallResponse{
			"map-a": {
				GenericResponse: types.GenericResponse{
					Status:  types.ResponseError,
					Message: "checksum failed",
				},
				ErrorType: types.InstallErrorChecksumFailed,
			},
		}),
		mockUninstallResponse(types.AssetTypeMap, nil, nil),
	))

	require.Empty(t, errs)
	require.Len(t, assetsToPurge, 1)
	require.Equal(t, types.AssetTypeMap, assetsToPurge[0].assetType)
	require.Equal(t, "map-a", assetsToPurge[0].assetID)
	require.Equal(t, "1.0.1", assetsToPurge[0].expectedVersion)
	require.Equal(t, types.InstallErrorChecksumFailed, assetsToPurge[0].errorCode)
}

func TestApplyPurgeOperations(t *testing.T) {
	testCases := []struct {
		name        string
		state       types.UserProfilesState
		candidates  []assetPurgeArgs
		expectOps   int
		expectErrs  int
		assertState func(t *testing.T, svc *UserProfiles)
	}{
		{
			name: "Checksum candidate removes matching map subscription",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.1"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			candidates: []assetPurgeArgs{
				{
					assetType:       types.AssetTypeMap,
					assetID:         "map-a",
					expectedVersion: "1.0.1",
					errorCode:       types.InstallErrorChecksumFailed,
				},
			},
			expectOps:  1,
			expectErrs: 0,
			assertState: func(t *testing.T, svc *UserProfiles) {
				t.Helper()
				active := svc.GetActiveProfile()
				_, exists := active.Profile.Subscriptions.Maps["map-a"]
				require.False(t, exists)
			},
		},
		{
			name: "Stale candidate version does not purge",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.2"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			candidates: []assetPurgeArgs{
				{
					assetType:       types.AssetTypeMap,
					assetID:         "map-a",
					expectedVersion: "1.0.1",
					errorCode:       types.InstallErrorInvalidManifest,
				},
			},
			expectOps:  0,
			expectErrs: 0,
			assertState: func(t *testing.T, svc *UserProfiles) {
				t.Helper()
				active := svc.GetActiveProfile()
				require.Equal(t, "1.0.2", active.Profile.Subscriptions.Maps["map-a"])
			},
		},
		{
			name: "Single pass purges map and mod",
			state: func() types.UserProfilesState {
				state := types.InitialProfilesState()
				profile := state.Profiles[types.DefaultProfileID]
				profile.Subscriptions.Maps["map-a"] = "1.0.1"
				profile.Subscriptions.Mods["mod-b"] = "2.0.0"
				state.Profiles[types.DefaultProfileID] = profile
				return state
			}(),
			candidates: []assetPurgeArgs{
				{
					assetType:       types.AssetTypeMap,
					assetID:         "map-a",
					expectedVersion: "1.0.1",
					errorCode:       types.InstallErrorInvalidArchive,
				},
				{
					assetType:       types.AssetTypeMod,
					assetID:         "mod-b",
					expectedVersion: "2.0.0",
					errorCode:       types.InstallErrorChecksumFailed,
				},
			},
			expectOps:  2,
			expectErrs: 0,
			assertState: func(t *testing.T, svc *UserProfiles) {
				t.Helper()
				active := svc.GetActiveProfile()
				_, mapExists := active.Profile.Subscriptions.Maps["map-a"]
				_, modExists := active.Profile.Subscriptions.Mods["mod-b"]
				require.False(t, mapExists)
				require.False(t, modExists)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.NewHarness(t)
			svc := loadedUserProfilesService(t, tc.state)
			operations, errs := svc.applyPurgeOperations(types.DefaultProfileID, tc.candidates)
			require.Lenf(t, operations, tc.expectOps, "ops=%+v errs=%+v", operations, errs)
			require.Lenf(t, errs, tc.expectErrs, "ops=%+v errs=%+v", operations, errs)
			if tc.assertState != nil {
				tc.assertState(t, svc)
			}

			persisted, err := ReadUserProfilesState()
			require.NoError(t, err)
			activeID := persisted.ActiveProfileID
			activeProfile := persisted.Profiles[activeID]
			liveProfile := svc.GetActiveProfile().Profile
			require.Equal(t, liveProfile.Subscriptions.Maps, activeProfile.Subscriptions.Maps)
			require.Equal(t, liveProfile.Subscriptions.Mods, activeProfile.Subscriptions.Mods)
		})
	}
}

// This test is intentionally concise given the Maps behavior is nearly identical
func TestSyncAssetSubscriptionsInstallDecisionsMods(t *testing.T) {
	installCalls := 0
	uninstallCalls := 0
	_, errs, _, _ := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockModAssetSyncArgs(assetSyncTestFixture{
		subscriptions: map[string]string{
			"mod-a": "1.0.1",
		},
		installedVersion: map[string]string{
			"mod-a": "1.0.0",
		},
		availableVersions: map[string]map[string]struct{}{
			"mod-a": {
				"1.0.1": {},
			},
		},
	},
		mockInstallResponse(types.AssetTypeMod, &installCalls, nil),
		mockUninstallResponse(types.AssetTypeMod, &uninstallCalls, nil),
	))

	require.Empty(t, errs)
	require.Equal(t, 1, installCalls)
	require.Equal(t, 1, uninstallCalls)
}

func TestSyncAssetSubscriptionsStopsWhenSnapshotIsStale(t *testing.T) {
	installCalls := 0
	uninstallCalls := 0
	args := mockMapAssetSyncArgs(
		assetSyncTestFixture{
			subscriptions: map[string]string{
				"map-a": "1.0.1",
			},
			installedVersion: map[string]string{
				"map-a": "1.0.0",
			},
			availableVersions: map[string]map[string]struct{}{
				"map-a": {
					"1.0.1": {},
				},
			},
		},
		mockInstallResponse(types.AssetTypeMap, &installCalls, nil),
		mockUninstallResponse(types.AssetTypeMap, &uninstallCalls, nil),
	)
	args.isStale = func() bool { return true }

	operations, errs, purgeCandidates, stale := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, args)
	require.True(t, stale)
	require.Empty(t, operations)
	require.Empty(t, errs)
	require.Empty(t, purgeCandidates)
	require.Equal(t, 0, installCalls)
	require.Equal(t, 0, uninstallCalls)
}
