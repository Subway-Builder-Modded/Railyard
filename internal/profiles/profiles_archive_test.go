package profiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"railyard/internal/paths"
	"railyard/internal/testutil"

	"github.com/stretchr/testify/require"
)

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
	require.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestClearRestoreDestinationRemovesDirectory(t *testing.T) {
	testutil.NewHarness(t)

	dirPath := filepath.Join(t.TempDir(), "restore-destination")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirPath, "data.txt"), []byte("x"), 0o644))

	require.NoError(t, clearRestoreDestination(dirPath))
	_, err := os.Stat(dirPath)
	require.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestClearRestoreDestinationNoopWhenMissing(t *testing.T) {
	testutil.NewHarness(t)

	dirPath := filepath.Join(t.TempDir(), "missing-destination")
	require.NoError(t, clearRestoreDestination(dirPath))
}
