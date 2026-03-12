package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"railyard/internal/config"
	"railyard/internal/downloader"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/testutil"
	"railyard/internal/testutil/registrytest"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func testUserProfilesLogger(t *testing.T) logger.Logger {
	t.Helper()
	return logger.LoggerAtPath(filepath.Join(t.TempDir(), "user_profiles_test.log"))
}

func requireProfileErrorType(t *testing.T, errs []types.UserProfilesError, expected types.UserProfilesErrorType) {
	t.Helper()
	require.NotEmpty(t, errs)
	require.Equal(t, expected, errs[0].ErrorType)
}

func userProfilesService(t *testing.T) *UserProfiles {
	t.Helper()
	svc, _, _ := userProfilesServiceWithDependencies(t)
	return svc
}

func loadedUserProfilesService(t *testing.T, state types.UserProfilesState) *UserProfiles {
	t.Helper()
	require.NoError(t, WriteUserProfilesState(state))

	svc, _, _ := userProfilesServiceWithDependencies(t)
	loadResult := svc.LoadProfiles()
	require.Equal(t, types.ResponseSuccess, loadResult.Status)
	return svc
}

func userProfilesServiceWithDependencies(t *testing.T) (*UserProfiles, *config.Config, *registry.Registry) {
	t.Helper()
	cfg := config.NewConfig()
	l := testUserProfilesLogger(t)
	reg := registry.NewRegistry(l)
	dl := downloader.NewDownloader(cfg, reg, l)
	return NewUserProfiles(reg, dl, l, cfg), cfg, reg
}

func loadedUserProfilesServiceWithDependencies(t *testing.T, state types.UserProfilesState) (*UserProfiles, *config.Config, *registry.Registry) {
	t.Helper()
	require.NoError(t, WriteUserProfilesState(state))

	svc, cfg, reg := userProfilesServiceWithDependencies(t)
	loadResult := svc.LoadProfiles()
	require.Equal(t, types.ResponseSuccess, loadResult.Status)
	return svc, cfg, reg
}

func writeRawUserProfilesFile(t *testing.T, content string) {
	t.Helper()

	path := paths.UserProfilesPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func newTestUserProfile(id string, name string) types.UserProfile {
	profile := types.DefaultProfile()
	profile.ID = id
	profile.Name = name
	return profile
}

type registryFixture struct {
	assetID      string
	assetType    types.AssetType
	versions     []string
	mapCode      string
	failVersions bool
}

func configureConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	exePath := filepath.Join(t.TempDir(), "subway-builder.exe")
	require.NoError(t, os.WriteFile(exePath, []byte("exe"), 0o644))
	cfg.Cfg.ExecutablePath = exePath
}

func mockRegistry(t *testing.T, reg *registry.Registry, fixtures []registryFixture) func() {
	sharedFixtures := make([]registrytest.UpdateFixture, 0, len(fixtures))
	for _, f := range fixtures {
		sharedFixtures = append(sharedFixtures, registrytest.UpdateFixture{
			AssetID:      f.assetID,
			AssetType:    f.assetType,
			Versions:     f.versions,
			MapCode:      f.mapCode,
			FailVersions: f.failVersions,
		})
	}
	return registrytest.MockRegistryServer(t, reg, sharedFixtures)
}

type assetSyncTestFixture struct {
	subscriptions     map[string]string
	installedVersion  map[string]string
	availableVersions map[string]map[string]struct{}
}

// TODO: Let's make this a function within the profiles.go so that we don't have to invoke this both in the main file and the test file...
func mockMapAssetSyncArgs(fixture assetSyncTestFixture, install func(string, string) types.GenericResponse, uninstall func(string) types.GenericResponse) assetSyncArgs[types.InstalledMapInfo, types.MapManifest] {
	return assetSyncArgs[types.InstalledMapInfo, types.MapManifest]{
		assetType:     types.AssetTypeMap,
		subscriptions: fixture.subscriptions,
		installedArgs: installedVersionArgs[types.InstalledMapInfo]{
			getInstalledAssetsFn: func() []types.InstalledMapInfo {
				items := make([]types.InstalledMapInfo, 0, len(fixture.installedVersion))
				for id, version := range fixture.installedVersion {
					items = append(items, types.InstalledMapInfo{ID: id, Version: version})
				}
				return items
			},
			idFn:      func(item types.InstalledMapInfo) string { return item.ID },
			versionFn: func(item types.InstalledMapInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.MapManifest]{
			getManifestsFn: func() []types.MapManifest {
				manifests := make([]types.MapManifest, 0, len(fixture.availableVersions))
				for assetID := range fixture.availableVersions {
					manifests = append(manifests, types.MapManifest{
						ID:     assetID,
						Update: types.UpdateConfig{Type: "custom", URL: assetID},
					})
				}
				return manifests
			},
			idFn:           func(item types.MapManifest) string { return item.ID },
			updateTypeFn:   func(item types.MapManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.MapManifest) string { return item.Update.URL },
			getVersionsFn: func(_ string, repoOrURL string) ([]types.VersionInfo, error) {
				versions := fixture.availableVersions[repoOrURL]
				list := make([]types.VersionInfo, 0, len(versions))
				for version := range versions {
					list = append(list, types.VersionInfo{Version: version})
				}
				return list, nil
			},
		},
		install:   install,
		uninstall: uninstall,
	}
}

// TODO: Let's make this a function within the profiles.go so that we don't have to invoke this both in the main file and the test file...
func mockModAssetSyncArgs(fixture assetSyncTestFixture, install func(string, string) types.GenericResponse, uninstall func(string) types.GenericResponse) assetSyncArgs[types.InstalledModInfo, types.ModManifest] {
	return assetSyncArgs[types.InstalledModInfo, types.ModManifest]{
		assetType:     types.AssetTypeMod,
		subscriptions: fixture.subscriptions,
		installedArgs: installedVersionArgs[types.InstalledModInfo]{
			getInstalledAssetsFn: func() []types.InstalledModInfo {
				items := make([]types.InstalledModInfo, 0, len(fixture.installedVersion))
				for id, version := range fixture.installedVersion {
					items = append(items, types.InstalledModInfo{ID: id, Version: version})
				}
				return items
			},
			idFn:      func(item types.InstalledModInfo) string { return item.ID },
			versionFn: func(item types.InstalledModInfo) string { return item.Version },
		},
		availableArgs: availableVersionArgs[types.ModManifest]{
			getManifestsFn: func() []types.ModManifest {
				manifests := make([]types.ModManifest, 0, len(fixture.availableVersions))
				for assetID := range fixture.availableVersions {
					manifests = append(manifests, types.ModManifest{
						ID:     assetID,
						Update: types.UpdateConfig{Type: "custom", URL: assetID},
					})
				}
				return manifests
			},
			idFn:           func(item types.ModManifest) string { return item.ID },
			updateTypeFn:   func(item types.ModManifest) string { return item.Update.Type },
			updateSourceFn: func(item types.ModManifest) string { return item.Update.URL },
			getVersionsFn: func(_ string, repoOrURL string) ([]types.VersionInfo, error) {
				versions := fixture.availableVersions[repoOrURL]
				list := make([]types.VersionInfo, 0, len(versions))
				for version := range versions {
					list = append(list, types.VersionInfo{Version: version})
				}
				return list, nil
			},
		},
		install:   install,
		uninstall: uninstall,
	}
}

func TestLoadProfilesBootstrapsAndPersistsStateWhenMissing(t *testing.T) {
	testutil.NewHarness(t)

	svc := userProfilesService(t)
	loadResult := svc.LoadProfiles()
	require.Equal(t, types.ResponseSuccess, loadResult.Status)
	require.Equal(t, types.DefaultProfileID, loadResult.Profile.ID)
	require.Equal(t, types.DefaultProfileName, loadResult.Profile.Name)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, types.DefaultProfileID, persisted.ActiveProfileID)

	defaultProfile, ok := persisted.Profiles[types.DefaultProfileID]
	require.True(t, ok)
	require.Equal(t, types.DefaultProfileID, defaultProfile.ID)
	require.Equal(t, types.DefaultProfileName, defaultProfile.Name)
	require.NotEmpty(t, defaultProfile.UUID)
}

func TestResolveActiveProfileFailsWhenNotLoaded(t *testing.T) {
	testutil.NewHarness(t)

	svc := userProfilesService(t)
	activeResult := svc.GetActiveProfile()
	require.Equal(t, types.ResponseError, activeResult.Status)
	requireProfileErrorType(t, activeResult.Errors, types.ErrorProfilesNotLoaded)
}

func TestLoadProfilesReturnsErrorForInvalidState(t *testing.T) {
	testutil.NewHarness(t)

	invalid := types.UserProfilesState{
		ActiveProfileID: "custom",
		Profiles: map[string]types.UserProfile{
			"custom": newTestUserProfile("custom", "Custom"),
		},
	}
	require.NoError(t, WriteUserProfilesState(invalid))

	svc := userProfilesService(t)
	loadResult := svc.LoadProfiles()
	require.Equal(t, types.ResponseError, loadResult.Status)
	require.NotEmpty(t, loadResult.Errors)
}

func TestResolveActiveProfileReturnsLoadedActiveProfile(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("custom", "Custom")
	state.ActiveProfileID = custom.ID
	state.Profiles[custom.ID] = custom
	require.NoError(t, WriteUserProfilesState(state))

	svc := userProfilesService(t)
	loadedActive := svc.LoadProfiles()
	require.Equal(t, types.ResponseSuccess, loadedActive.Status)
	require.Equal(t, custom.ID, loadedActive.Profile.ID)
	require.Equal(t, custom.Name, loadedActive.Profile.Name)

	active := svc.GetActiveProfile()
	require.Equal(t, types.ResponseSuccess, active.Status)
	require.Equal(t, custom.ID, active.Profile.ID)
	require.Equal(t, custom.Name, active.Profile.Name)
}

func TestQuarantineUserProfilesFileMovesSourceToBackup(t *testing.T) {
	testutil.NewHarness(t)
	writeRawUserProfilesFile(t, "{}")

	svc := userProfilesService(t)
	success, backupPath := svc.QuarantineUserProfiles()
	require.True(t, success)
	require.NotEmpty(t, backupPath)
	require.True(t, strings.Contains(filepath.Base(backupPath), "user_profiles.invalid."))

	_, err := os.Stat(backupPath)
	require.NoError(t, err)

	_, err = os.Stat(paths.UserProfilesPath())
	require.True(t, os.IsNotExist(err))
}

func TestUpdateSubscriptionsSubscribeMapAddsOperationAndPersistsByDefault(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-a": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
		ForceSync: false,
	}

	result := svc.UpdateSubscriptions(req)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.True(t, result.Persisted)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.Maps["map-a"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, "map-a", result.Operations[0].AssetID)
	require.Equal(t, types.AssetTypeMap, result.Operations[0].Type)
	require.Equal(t, types.SubscriptionActionSubscribe, result.Operations[0].Action)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, "1.2.3", persisted.Profiles[types.DefaultProfileID].Subscriptions.Maps["map-a"])
}

func TestUpdateSubscriptionsForceSyncPersistsStateAndSyncs(t *testing.T) {
	testutil.NewHarness(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Mods["mod-a"] = "2.0.0"
	state.Profiles[types.DefaultProfileID] = profile

	svc, cfg, reg := loadedUserProfilesServiceWithDependencies(t, state)
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	reg.AddInstalledMod("mod-a", "2.0.0")

	req := types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionUnsubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"mod-a": {Type: types.AssetTypeMod},
		},
		ForceSync: true,
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
		Assets: map[string]types.SubscriptionUpdateItem{
			"missing": {Type: types.AssetTypeMap},
		},
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "Subscriptions updated", result.Message)
	require.Empty(t, result.Errors)
	require.Empty(t, result.Operations)
}

func TestUpdateSubscriptionsRejectsInvalidRequests(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: "missing",
		Action:    types.SubscriptionActionSubscribe,
	})
	requireProfileErrorType(t, result.Errors, types.ErrorProfileNotFound)
	require.Equal(t, types.ResponseError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Equal(t, types.ErrorProfileNotFound, result.Errors[0].ErrorType)
	require.Equal(t, "missing", result.Errors[0].ProfileID)

	result = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionAction("bad-action"),
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
}

func TestUpdateSubscriptionsAcceptsSemverVersionString(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
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
		ForceSync: true,
	})

	require.Equal(t, types.ResponseError, result.Status)
	require.Equal(t, "Failed to sync subscriptions", result.Message)
	require.NotEmpty(t, result.Errors)
	require.Equal(t, types.ErrorLookupFailed, result.Errors[len(result.Errors)-1].ErrorType)
	require.Equal(t, types.DefaultProfileID, result.Errors[len(result.Errors)-1].ProfileID)
}

func TestUpdateAllSubscriptionsToLatest(t *testing.T) {
	type expectation struct {
		expectedStatus          types.Status
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
		state        types.UserProfilesState
		setup        func(t *testing.T, cfg *config.Config, reg *registry.Registry) func()
		expected     expectation
		assertResult func(t *testing.T, svc *UserProfiles, reg *registry.Registry, result types.UpdateSubscriptionsResult)
	}{
		{
			name:      "Updates map and mod to latest semver and syncs",
			profileID: types.DefaultProfileID,
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
				expectedStatus:          types.ResponseSuccess,
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
				reg.AddInstalledMod("mod-missing", "1.0.0")
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
				expectedStatus:          types.ResponseWarn,
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
				expectedStatus:          types.ResponseError,
				expectedPersisted:       true, // state is updated to desired but sync fails
				expectedOperationByID:   map[string]string{"map-a": "1.1.0"},
				expectedMapSubscription: "1.1.0",
				expectedErrContains:     `Failed sync action: subscribe map "map-a" failed`,
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

			result := svc.UpdateAllSubscriptionsToLatest(tc.profileID)
			require.Equal(t, tc.expected.expectedStatus, result.Status)
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

func TestSyncActionErrorIgnoresWarnings(t *testing.T) {
	t.Run("Duplicate install warning returns no error", func(t *testing.T) {
		err := syncActionError(
			types.SubscriptionActionSubscribe,
			types.AssetTypeMap,
			"map-a",
			types.GenericResponse{Status: types.ResponseWarn, Message: "Duplicate request skipped: install already queued"},
		)
		require.NoError(t, err)
	})

	t.Run("Duplicate uninstall warning returns no error", func(t *testing.T) {
		err := syncActionError(
			types.SubscriptionActionUnsubscribe,
			types.AssetTypeMod,
			"mod-a",
			types.GenericResponse{Status: types.ResponseWarn, Message: "Duplicate request skipped: uninstall already queued"},
		)
		require.NoError(t, err)
	})

	t.Run("Non-duplicate warning still returns error", func(t *testing.T) {
		err := syncActionError(
			types.SubscriptionActionSubscribe,
			types.AssetTypeMap,
			"map-a",
			types.GenericResponse{Status: types.ResponseWarn, Message: "Map with ID map-a is not currently installed. No action taken."},
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

		prepare func(t *testing.T, cfg *config.Config, reg *registry.Registry) func()

		expectedErrors []string
		expectedState  expectedState
	}{
		{
			name:  "No subscriptions with no installed assets is no-op",
			state: types.InitialProfilesState(),
			expectedState: expectedState{
				mods: nil,
				maps: nil,
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
			for _, mod := range tc.initialMods {
				reg.AddInstalledMod(mod.ID, mod.Version)
			}
			for _, m := range tc.initialMaps {
				reg.AddInstalledMap(m.ID, m.Version, m.MapConfig)
			}
			var cleanup func()
			if tc.prepare != nil {
				cleanup = tc.prepare(t, cfg, reg)
			}
			if cleanup != nil {
				defer cleanup()
			}

			result := svc.SyncSubscriptions(types.DefaultProfileID)
			if len(tc.expectedErrors) == 0 {
				require.Equal(t, types.ResponseSuccess, result.Status)
			} else {
				require.Equal(t, types.ResponseError, result.Status)
				for _, expected := range tc.expectedErrors {
					found := false
					for _, profileErr := range result.Errors {
						if strings.Contains(profileErr.Error(), expected) {
							found = true
							break
						}
					}
					require.True(t, found)
				}
			}

			require.Equal(t, tc.expectedState.mods, reg.GetInstalledMods())
			require.Equal(t, tc.expectedState.maps, reg.GetInstalledMaps())
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
			_, errs := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockMapAssetSyncArgs(assetSyncTestFixture{
				subscriptions:     tc.subscriptions,
				installedVersion:  tc.installedVersion,
				availableVersions: tc.availableVersions,
			}, func(assetID string, version string) types.GenericResponse {
				installCalls++
				return types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"}
			}, func(assetID string) types.GenericResponse {
				uninstallCalls++
				return types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"}
			}))

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
	_, errs := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockMapAssetSyncArgs(
		fixture,
		func(assetID string, version string) types.GenericResponse {
			installCalls++
			return types.GenericResponse{
				Status:  types.ResponseError,
				Message: "Failed to extract map zip: Cannot install map because its code matches a vanilla map included with the game or an already installed map.",
			}
		},
		func(assetID string) types.GenericResponse {
			uninstallCalls++
			return types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"}
		},
	))

	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "Failed to extract map zip")
	require.Equal(t, 1, installCalls)
	require.Equal(t, 1, uninstallCalls)
}

// This test is intentionally concise given the Maps behavior is nearly identical
func TestSyncAssetSubscriptionsInstallDecisionsMods(t *testing.T) {
	installCalls := 0
	uninstallCalls := 0
	_, errs := syncAssetSubscriptions(testUserProfilesLogger(t), types.DefaultProfileID, mockModAssetSyncArgs(assetSyncTestFixture{
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
	}, func(assetID string, version string) types.GenericResponse {
		installCalls++
		return types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"}
	}, func(assetID string) types.GenericResponse {
		uninstallCalls++
		return types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"}
	}))

	require.Empty(t, errs)
	require.Equal(t, 1, installCalls)
	require.Equal(t, 1, uninstallCalls)
}

func TestUpdateUIPreferences(t *testing.T) {
	testutil.NewHarness(t)

	svc := loadedUserProfilesService(t, types.InitialProfilesState())
	result := svc.UpdateUIPreferences(types.UIPreferences{
		Theme:          types.ThemeLight,
		DefaultPerPage: types.PageSize24,
	})

	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, types.ThemeLight, result.Profile.UIPreferences.Theme)
	require.Equal(t, types.PageSize24, result.Profile.UIPreferences.DefaultPerPage)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, types.ThemeLight, persisted.Profiles[persisted.ActiveProfileID].UIPreferences.Theme)
	require.Equal(t, types.PageSize24, persisted.Profiles[persisted.ActiveProfileID].UIPreferences.DefaultPerPage)
}

func TestUpdateUIPreferencesRejectsInvalid(t *testing.T) {
	testutil.NewHarness(t)

	svc := loadedUserProfilesService(t, types.InitialProfilesState())
	result := svc.UpdateUIPreferences(types.UIPreferences{
		Theme:          types.ThemeMode("retro"),
		DefaultPerPage: types.PageSize(30),
	})

	require.Equal(t, types.ResponseError, result.Status)
	requireProfileErrorType(t, result.Errors, types.ErrorUnknown)

	active := svc.GetActiveProfile()
	require.Equal(t, types.ThemeDark, active.Profile.UIPreferences.Theme)
	require.Equal(t, types.PageSize12, active.Profile.UIPreferences.DefaultPerPage)
}
