package profiles

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"railyard/internal/config"
	"railyard/internal/downloader"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/registry"
	"railyard/internal/types"

	"net/http"
	"net/http/httptest"
	"reflect"

	"github.com/stretchr/testify/require"
)

func setEnv(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("APPDATA", root)
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("ProgramFiles", root)
	t.Setenv("ProgramFiles(x86)", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", root)
}

func testUserProfilesLogger(t *testing.T) logger.Logger {
	t.Helper()
	return logger.LoggerAtPath(filepath.Join(t.TempDir(), "user_profiles_test.log"))
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
	_, err := svc.LoadProfiles()
	require.NoError(t, err)
	return svc
}

func userProfilesServiceWithDependencies(t *testing.T) (*UserProfiles, *config.Config, *registry.Registry) {
	t.Helper()
	cfg := config.NewConfig()
	reg := registry.NewRegistry()
	l := testUserProfilesLogger(t)
	dl := downloader.NewDownloader(cfg, reg, l)
	return NewUserProfiles(reg, dl, l), cfg, reg
}

func loadedUserProfilesServiceWithDependencies(t *testing.T, state types.UserProfilesState) (*UserProfiles, *config.Config, *registry.Registry) {
	t.Helper()
	require.NoError(t, WriteUserProfilesState(state))

	svc, cfg, reg := userProfilesServiceWithDependencies(t)
	_, err := svc.LoadProfiles()
	require.NoError(t, err)
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

type syncAssetAvailabilityFixture struct {
	assetID   string
	version   string
	assetType types.AssetType
	mapCode   string
}

func setRegistryManifestsForTest(t *testing.T, reg *registry.Registry, mods []types.ModManifest, maps []types.MapManifest) {
	t.Helper()
	setUnexportedField(t, reg, "mods", mods)
	setUnexportedField(t, reg, "maps", maps)
}

func setUnexportedField(t *testing.T, target any, fieldName string, value any) {
	t.Helper()
	rv := reflect.ValueOf(target)
	require.True(t, rv.Kind() == reflect.Ptr, "target must be pointer")
	elem := rv.Elem()
	field := elem.FieldByName(fieldName)
	require.True(t, field.IsValid(), "field %q not found", fieldName)

	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

// TODO: Create a global test helper to share these across different services that depend on Registry
func mockZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	tempZip := filepath.Join(t.TempDir(), "fixture.zip")
	f, err := os.Create(tempZip)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	for name, content := range files {
		entry, createErr := w.Create(name)
		require.NoError(t, createErr)
		_, writeErr := entry.Write(content)
		require.NoError(t, writeErr)
	}
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	data, err := os.ReadFile(tempZip)
	require.NoError(t, err)
	return data
}

func mockMod(t *testing.T) []byte {
	t.Helper()
	return mockZip(t, map[string][]byte{
		"README.txt": []byte("mod fixture"),
	})
}

func mockMap(t *testing.T, code string) []byte {
	t.Helper()
	configJSON, err := json.Marshal(types.ConfigData{
		Code: code,
		Name: "Fixture Map",
	})
	require.NoError(t, err)

	return mockZip(t, map[string][]byte{
		"config.json":              configJSON,
		"demand_data.json":         []byte("{}"),
		"roads.geojson":            []byte(`{"type":"FeatureCollection","features":[]}`),
		"runways_taxiways.geojson": []byte(`{"type":"FeatureCollection","features":[]}`),
		"buildings_index.json":     []byte("{}"),
		"tiles.pmtiles":            []byte("tiles"),
		"thumbnail.svg":            []byte("<svg></svg>"),
	})
}

func configureValidConfigForInstall(t *testing.T, cfg *config.Config) {
	t.Helper()
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	exePath := filepath.Join(t.TempDir(), "subway-builder.exe")
	require.NoError(t, os.WriteFile(exePath, []byte("exe"), 0o644))
	cfg.Cfg.ExecutablePath = exePath
}

func mockRegistryAvailability(t *testing.T, reg *registry.Registry, fixtures []syncAssetAvailabilityFixture) func() {
	t.Helper()
	zipByDownloadPath := map[string][]byte{}

	mods := []types.ModManifest{}
	maps := []types.MapManifest{}

	handler := http.NewServeMux()
	for _, fixture := range fixtures {
		current := fixture
		updatePath := "/updates/" + fixture.assetID + ".json"
		downloadPath := "/downloads/" + fixture.assetID + "-" + fixture.version + ".zip"

		var zipBytes []byte
		if fixture.assetType == types.AssetTypeMap {
			zipBytes = mockMap(t, fixture.mapCode)
			maps = append(maps, types.MapManifest{
				ID:     fixture.assetID,
				Update: types.UpdateConfig{Type: "custom", URL: "{{BASE_URL}}" + updatePath},
			})
		} else {
			zipBytes = mockMod(t)
			mods = append(mods, types.ModManifest{
				ID:     fixture.assetID,
				Update: types.UpdateConfig{Type: "custom", URL: "{{BASE_URL}}" + updatePath},
			})
		}
		zipByDownloadPath[downloadPath] = zipBytes

		handler.HandleFunc(updatePath, func(w http.ResponseWriter, r *http.Request) {
			updatePayload := types.CustomUpdateFile{
				SchemaVersion: 1,
				Versions: []types.CustomUpdateVersion{
					{
						Version:  current.version,
						Download: "http://" + r.Host + downloadPath,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(updatePayload))
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if content, ok := zipByDownloadPath[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(content)
			return
		}
		handler.ServeHTTP(w, r)
	}))

	for i := range mods {
		mods[i].Update.URL = strings.ReplaceAll(mods[i].Update.URL, "{{BASE_URL}}", server.URL)
	}
	for i := range maps {
		maps[i].Update.URL = strings.ReplaceAll(maps[i].Update.URL, "{{BASE_URL}}", server.URL)
	}

	setRegistryManifestsForTest(t, reg, mods, maps)
	return server.Close
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
	setEnv(t)

	svc := userProfilesService(t)
	active, err := svc.LoadProfiles()
	require.NoError(t, err)
	require.Equal(t, types.DefaultProfileID, active.ID)
	require.Equal(t, types.DefaultProfileName, active.Name)

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
	setEnv(t)

	svc := userProfilesService(t)
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
	require.NoError(t, WriteUserProfilesState(invalid))

	svc := userProfilesService(t)
	_, err := svc.LoadProfiles()
	require.ErrorIs(t, err, types.ErrInvalidState)
}

func TestResolveActiveProfileReturnsLoadedActiveProfile(t *testing.T) {
	setEnv(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("custom", "Custom")
	state.ActiveProfileID = custom.ID
	state.Profiles[custom.ID] = custom
	require.NoError(t, WriteUserProfilesState(state))

	svc := userProfilesService(t)
	loadedActive, err := svc.LoadProfiles()
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

func TestUpdateSubscriptionsSubscribeMapAddsOperationAndRuntimeOnlyByDefault(t *testing.T) {
	setEnv(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

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

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Empty(t, persisted.Profiles[types.DefaultProfileID].Subscriptions.Maps)
}

func TestUpdateSubscriptionsForceSyncPersistsStateAndSyncs(t *testing.T) {
	setEnv(t)
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

	result, err := svc.UpdateSubscriptions(req)
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
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
	setEnv(t)
	state := types.InitialProfilesState()
	profile := state.Profiles[types.DefaultProfileID]
	profile.Subscriptions.Maps["map-a"] = "1.2.3"
	state.Profiles[types.DefaultProfileID] = profile
	svc := loadedUserProfilesService(t, state)

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
	svc := loadedUserProfilesService(t, state)

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
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

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
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

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

	_, err = svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"asset": {Type: types.AssetTypeMap, Version: types.Version("not-semver")},
		},
	})
	require.ErrorIs(t, err, ErrInvalidVersion)
}

func TestUpdateSubscriptionsAcceptsSemverVersionString(t *testing.T) {
	setEnv(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result, err := svc.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
		ProfileID: types.DefaultProfileID,
		Action:    types.SubscriptionActionSubscribe,
		Assets: map[string]types.SubscriptionUpdateItem{
			"map-x": {Type: types.AssetTypeMap, Version: types.Version("1.2.3")},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "subscriptions updated", result.Message)
	require.Equal(t, "1.2.3", result.Profile.Subscriptions.Maps["map-x"])
	require.Len(t, result.Operations, 1)
	require.Equal(t, types.Version("1.2.3"), result.Operations[0].Version)
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
			name:  "no subscriptions with no installed assets is no-op",
			state: types.InitialProfilesState(),
			expectedState: expectedState{
				mods: nil,
				maps: nil,
			},
		},
		{
			name:  "unsubscribed installed mod is removed",
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
			name:  "unsubscribed installed map is removed",
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
			name: "sync errors when subscribed assets are unavailable",
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
				`subscribe mod "mod-b" failed`,
				`subscribe map "map-b" failed`,
			},
			expectedState: expectedState{
				mods: []types.InstalledModInfo{},
				maps: []types.InstalledMapInfo{},
			},
		},
		{
			name: "sync succeeds when subscribed assets are available",
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
				configureValidConfigForInstall(t, cfg)
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return mockRegistryAvailability(t, reg, []syncAssetAvailabilityFixture{
					{assetID: "mod-b", version: "1.0.0", assetType: types.AssetTypeMod},
					{assetID: "map-b", version: "1.0.0", assetType: types.AssetTypeMap, mapCode: "BBB"},
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
			name: "sync errors on attempted update to new version of asset without availability",
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
				`subscribe map "map-a" failed`,
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
			name: "sync succeeds on attempted update to new version of asset with availability",
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
				configureValidConfigForInstall(t, cfg)
				tilePath := filepath.Join(paths.AppDataRoot(), "tiles", "AAA.pmtiles")
				require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
				require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
				return mockRegistryAvailability(t, reg, []syncAssetAvailabilityFixture{
					{assetID: "map-a", version: "1.0.1", assetType: types.AssetTypeMap, mapCode: "AAA"},
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
			setEnv(t)

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

			err := svc.SyncSubscriptions(types.DefaultProfileID)
			if len(tc.expectedErrors) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expected := range tc.expectedErrors {
					require.Contains(t, err.Error(), expected)
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
			name: "already installed version skips install even when unavailable",
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
			name: "available newer version triggers install and updates index",
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
			name: "unavailable version blocks install",
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
				`subscribe map "map-a" failed: version "2.0.0" is not available`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			installCalls := 0
			uninstallCalls := 0
			errs := syncAssetSubscriptions(mockMapAssetSyncArgs(assetSyncTestFixture{
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

// This test is intentionally concise given the Maps behavior is nearly identical
func TestSyncAssetSubscriptionsInstallDecisionsMods(t *testing.T) {
	installCalls := 0
	uninstallCalls := 0
	errs := syncAssetSubscriptions(mockModAssetSyncArgs(assetSyncTestFixture{
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
