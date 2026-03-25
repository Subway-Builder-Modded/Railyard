package profiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/testutil"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func findProfileErrorType(errs []types.UserProfilesError, wanted types.UserProfilesErrorType) bool {
	for _, item := range errs {
		if item.ErrorType == wanted {
			return true
		}
	}
	return false
}

func TestCreateProfileDefaultsAndPersistence(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.CreateProfile(types.CreateProfileRequest{Name: "My Profile"})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "profile_0", result.Profile.ID)
	require.Equal(t, "My Profile", result.Profile.Name)
	require.NotEmpty(t, result.Profile.UUID)

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	require.Equal(t, types.DefaultProfileID, persisted.ActiveProfileID)
	require.Len(t, persisted.Profiles, 2)
	created, ok := persisted.Profiles[result.Profile.ID]
	require.True(t, ok)
	require.Equal(t, "My Profile", created.Name)
}

func TestCreateProfileRejectsDuplicateNameCaseInsensitive(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.CreateProfile(types.CreateProfileRequest{Name: " default "})
	require.Equal(t, types.ResponseError, result.Status)
	require.True(t, findProfileErrorType(result.Errors, types.ErrorDuplicateName))
}

func TestCreateProfileUsesMaxExistingProfileSuffix(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	profileTwo := newTestUserProfile("profile_2", "P2")
	profileTen := newTestUserProfile("profile_10", "P10")
	custom := newTestUserProfile("custom_profile", "Custom")
	state.Profiles[profileTwo.ID] = profileTwo
	state.Profiles[profileTen.ID] = profileTen
	state.Profiles[custom.ID] = custom

	svc := loadedUserProfilesService(t, state)
	result := svc.CreateProfile(types.CreateProfileRequest{Name: "Next"})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, "profile_11", result.Profile.ID)
}

func TestDeleteProfileValidation(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	defaultDelete := svc.DeleteProfile(types.DefaultProfileID)
	require.Equal(t, types.ResponseError, defaultDelete.Status)
	require.True(t, findProfileErrorType(defaultDelete.Errors, types.ErrorDefaultProtected))

	missingDelete := svc.DeleteProfile("missing")
	require.Equal(t, types.ResponseError, missingDelete.Status)
	require.True(t, findProfileErrorType(missingDelete.Errors, types.ErrorProfileNotFound))
}

func TestDeleteProfileRejectsActive(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	custom := newTestUserProfile("profile_0", "Custom")
	state.Profiles[custom.ID] = custom
	state.ActiveProfileID = custom.ID

	svc := loadedUserProfilesService(t, state)
	result := svc.DeleteProfile(custom.ID)
	require.Equal(t, types.ResponseError, result.Status)
	require.True(t, findProfileErrorType(result.Errors, types.ErrorActiveProtected))
}

func TestDeleteProfileRemovesArchive(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	target := newTestUserProfile("profile_0", "Delete Me")
	state.Profiles[target.ID] = target

	svc := loadedUserProfilesService(t, state)
	archivePath := profileArchivePath(target.UUID)
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, []byte("archive"), 0o644))

	result := svc.DeleteProfile(target.ID)
	require.Equal(t, types.ResponseSuccess, result.Status)
	_, statErr := os.Stat(archivePath)
	require.True(t, errors.Is(statErr, fs.ErrNotExist))

	persisted, err := ReadUserProfilesState()
	require.NoError(t, err)
	_, exists := persisted.Profiles[target.ID]
	require.False(t, exists)
}

func TestSwapProfileMissingTargetFails(t *testing.T) {
	testutil.NewHarness(t)
	svc := loadedUserProfilesService(t, types.InitialProfilesState())

	result := svc.SwapProfile(types.SwapProfileRequest{ProfileID: "missing"})
	require.Equal(t, types.ResponseError, result.Status)
	require.True(t, findProfileErrorType(result.Errors, types.ErrorProfileNotFound))
}

func TestSwapProfileWarnsWithoutForceWhenTargetArchiveMissing(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	target := newTestUserProfile("profile_0", "Target")
	state.Profiles[target.ID] = target
	svc := loadedUserProfilesService(t, state)

	current := svc.GetActiveProfile().Profile
	result := svc.SwapProfile(types.SwapProfileRequest{ProfileID: target.ID})
	require.Equal(t, types.ResponseWarn, result.Status)
	require.True(t, findProfileErrorType(result.Errors, types.ErrorArchiveMissing))

	activeAfter := svc.GetActiveProfile()
	require.Equal(t, types.ResponseSuccess, activeAfter.Status)
	require.Equal(t, current.ID, activeAfter.Profile.ID)

	_, err := os.Stat(profileArchivePath(current.UUID))
	require.NoError(t, err)
}

func TestSwapProfileForceWithoutArchiveSwapsAndSyncs(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	target := newTestUserProfile("profile_0", "Target")
	state.Profiles[target.ID] = target
	svc := loadedUserProfilesService(t, state)

	result := svc.SwapProfile(types.SwapProfileRequest{
		ProfileID: target.ID,
		ForceSwap: true,
	})
	require.Equal(t, types.ResponseSuccess, result.Status)
	require.Equal(t, target.ID, result.Profile.ID)

	activeAfter := svc.GetActiveProfile()
	require.Equal(t, types.ResponseSuccess, activeAfter.Status)
	require.Equal(t, target.ID, activeAfter.Profile.ID)
}

func TestSwapProfileUsesFreshArchiveRestorePath(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	target := newTestUserProfile("profile_0", "Target")
	state.Profiles[target.ID] = target
	svc := loadedUserProfilesService(t, state)

	archiveResult := svc.CreateProfileArchive(target.ID)
	require.Equal(t, types.ResponseSuccess, archiveResult.Status)

	swapResult := svc.SwapProfile(types.SwapProfileRequest{ProfileID: target.ID})
	require.Equal(t, types.ResponseSuccess, swapResult.Status)
	require.Equal(t, target.ID, swapResult.Profile.ID)

	// Restore keeps existing behavior and deletes archive after successful restore.
	_, statErr := os.Stat(profileArchivePath(target.UUID))
	require.True(t, errors.Is(statErr, fs.ErrNotExist))
}

func TestProfileArchiveFreshnessMetadata(t *testing.T) {
	testutil.NewHarness(t)

	state := types.InitialProfilesState()
	target := newTestUserProfile("profile_0", "Target")
	target.Subscriptions.Maps["map-a"] = "1.0.0"
	target.Subscriptions.LocalMaps["local-map"] = "1.0.0"
	target.Subscriptions.Mods["mod-a"] = "1.0.0"
	state.Profiles[target.ID] = target

	svc := loadedUserProfilesService(t, state)
	archiveResult := svc.CreateProfileArchive(target.ID)
	require.Equal(t, types.ResponseSuccess, archiveResult.Status)

	fresh, err := svc.isProfileArchiveFresh(target)
	require.NoError(t, err)
	require.True(t, fresh)

	target.Subscriptions.Maps["map-a"] = "2.0.0"
	stale, err := svc.isProfileArchiveFresh(target)
	require.NoError(t, err)
	require.False(t, stale)
}
