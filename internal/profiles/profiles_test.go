package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/config"
	"railyard/internal/constants"
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
	cfg := config.NewConfig(testutil.TestLogSink{})
	l := testUserProfilesLogger(t)
	reg := registry.NewRegistry(l, cfg)
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
	assetID            string
	assetType          types.AssetType
	versions           []string
	mapCode            string
	failVersions       bool
	missingModManifest bool
}

func configureConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	exePath := filepath.Join(t.TempDir(), "subway-builder.exe")
	require.NoError(t, os.WriteFile(exePath, []byte("exe"), 0o755))
	cfg.Cfg.ExecutablePath = exePath
}

func materializeInstalledAssets(
	t *testing.T,
	cfg *config.Config,
	mods []types.InstalledModInfo,
	maps []types.InstalledMapInfo,
) {
	t.Helper()
	for _, mod := range mods {
		modPath := paths.JoinLocalPath(paths.MetroMakerModsPath(cfg.Cfg.MetroMakerDataPath), mod.ID)
		require.NoError(t, os.MkdirAll(modPath, 0o755))
		require.NoError(t, os.WriteFile(paths.JoinLocalPath(modPath, constants.RailyardAssetMarker), []byte(""), 0o644))
	}

	for _, m := range maps {
		mapPath := paths.JoinLocalPath(paths.MetroMakerMapsDataPath(cfg.Cfg.MetroMakerDataPath), m.MapConfig.Code)
		require.NoError(t, os.MkdirAll(mapPath, 0o755))
		require.NoError(t, os.WriteFile(paths.JoinLocalPath(mapPath, constants.RailyardAssetMarker), []byte(""), 0o644))

		tilePath := paths.JoinLocalPath(paths.AppDataRoot(), "tiles", m.MapConfig.Code+".pmtiles")
		require.NoError(t, os.MkdirAll(filepath.Dir(tilePath), 0o755))
		require.NoError(t, os.WriteFile(tilePath, []byte("tile"), 0o644))
	}
}

func mockRegistry(t *testing.T, reg *registry.Registry, fixtures []registryFixture) func() {
	sharedFixtures := make([]registrytest.UpdateFixture, 0, len(fixtures))
	for _, f := range fixtures {
		sharedFixtures = append(sharedFixtures, registrytest.UpdateFixture{
			AssetID:            f.assetID,
			AssetType:          f.assetType,
			Versions:           f.versions,
			MapCode:            f.mapCode,
			FailVersions:       f.failVersions,
			MissingModManifest: f.missingModManifest,
		})
	}
	return registrytest.MockRegistryServer(t, reg, sharedFixtures)
}

type assetSyncTestFixture struct {
	subscriptions     map[string]string
	installedVersion  map[string]string
	availableVersions map[string]map[string]struct{}
}

func mockInstallResponse(
	assetType types.AssetType,
	callCount *int,
	overrides map[string]types.AssetInstallResponse,
) func(string, string) types.AssetInstallResponse {
	return func(assetID string, version string) types.AssetInstallResponse {
		if callCount != nil {
			*callCount++
		}
		if override, ok := overrides[assetID]; ok {
			override.GenericResponse = types.GenericResponse{
				Status:  orDefault(override.Status, types.ResponseSuccess),
				Message: orDefault(override.Message, "ok"),
			}
			override.AssetType = orDefault(override.AssetType, assetType)
			override.AssetID = orDefault(override.AssetID, assetID)
			override.Version = orDefault(override.Version, version)
			return override
		}
		return types.AssetInstallResponse{
			GenericResponse: types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"},
			AssetType:       assetType,
			AssetID:         assetID,
			Version:         version,
		}
	}
}

func mockUninstallResponse(
	assetType types.AssetType,
	callCount *int,
	overrides map[string]types.AssetUninstallResponse,
) func(string) types.AssetUninstallResponse {
	return func(assetID string) types.AssetUninstallResponse {
		if callCount != nil {
			*callCount++
		}
		if override, ok := overrides[assetID]; ok {
			override.GenericResponse = types.GenericResponse{
				Status:  orDefault(override.Status, types.ResponseSuccess),
				Message: orDefault(override.Message, "ok"),
			}
			override.AssetType = orDefault(override.AssetType, assetType)
			override.AssetID = orDefault(override.AssetID, assetID)
			return override
		}
		return types.AssetUninstallResponse{
			GenericResponse: types.GenericResponse{Status: types.ResponseSuccess, Message: "ok"},
			AssetType:       assetType,
			AssetID:         assetID,
		}
	}
}

func orDefault[T comparable](value, fallback T) T {
	var zero T
	if value == zero {
		return fallback
	}
	return value
}

// TODO: Let's make this a function within the profiles.go so that we don't have to invoke this both in the main file and the test file...
func mockMapAssetSyncArgs(fixture assetSyncTestFixture, install func(string, string) types.AssetInstallResponse, uninstall func(string) types.AssetUninstallResponse) assetSyncArgs[types.InstalledMapInfo, types.MapManifest] {
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
func mockModAssetSyncArgs(fixture assetSyncTestFixture, install func(string, string) types.AssetInstallResponse, uninstall func(string) types.AssetUninstallResponse) assetSyncArgs[types.InstalledModInfo, types.ModManifest] {
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

func TestUpdateUIPreferences(t *testing.T) {
	testutil.NewHarness(t)

	svc := loadedUserProfilesService(t, types.InitialProfilesState())
	result := svc.UpdateUIPreferences(types.UIPreferences{
		Theme:          types.ThemeLight,
		DefaultPerPage: types.PageSize24,
		SearchViewMode: types.SearchViewModeFull,
	})

	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, types.ThemeLight, result.Profile.UIPreferences.Theme)
	require.Equal(t, types.PageSize24, result.Profile.UIPreferences.DefaultPerPage)
	require.Equal(t, types.SearchViewModeFull, result.Profile.UIPreferences.SearchViewMode)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, types.ThemeLight, persisted.Profiles[persisted.ActiveProfileID].UIPreferences.Theme)
	require.Equal(t, types.PageSize24, persisted.Profiles[persisted.ActiveProfileID].UIPreferences.DefaultPerPage)
	require.Equal(t, types.SearchViewModeFull, persisted.Profiles[persisted.ActiveProfileID].UIPreferences.SearchViewMode)
}

func TestUpdateUIPreferencesRejectsInvalid(t *testing.T) {
	testutil.NewHarness(t)

	svc := loadedUserProfilesService(t, types.InitialProfilesState())
	result := svc.UpdateUIPreferences(types.UIPreferences{
		Theme:          types.ThemeMode("retro"),
		DefaultPerPage: types.PageSize(30),
		SearchViewMode: types.SearchViewMode("abcdefg"),
	})

	require.Equal(t, types.ResponseError, result.Status)
	requireProfileErrorType(t, result.Errors, types.ErrorUnknown)

	active := svc.GetActiveProfile()
	require.Equal(t, types.ThemeDark, active.Profile.UIPreferences.Theme)
	require.Equal(t, types.PageSize12, active.Profile.UIPreferences.DefaultPerPage)
	require.Equal(t, types.SearchViewModeFull, active.Profile.UIPreferences.SearchViewMode)
}
