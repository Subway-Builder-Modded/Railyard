package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func configureAtomicStagingRootsForTest(t *testing.T, roots []StagingRoot) {
	t.Helper()
	ConfigureTmpStagingRoots(roots)
	t.Cleanup(func() {
		ConfigureTmpStagingRoots(nil)
	})
}

func TestWriteFilesAtomicallyWritesMultipleFiles(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.json")
	second := filepath.Join(root, "second.json")

	err := WriteFilesAtomically([]AtomicFileWrite{
		{
			Path:  first,
			Label: "first",
			Data:  []byte(`{"name":"a"}`),
			Perm:  0o644,
		},
		{
			Path:  second,
			Label: "second",
			Data:  []byte(`{"name":"b"}`),
			Perm:  0o644,
		},
	})
	require.NoError(t, err)

	firstData, firstErr := os.ReadFile(first)
	require.NoError(t, firstErr)
	require.JSONEq(t, `{"name":"a"}`, string(firstData))

	secondData, secondErr := os.ReadFile(second)
	require.NoError(t, secondErr)
	require.JSONEq(t, `{"name":"b"}`, string(secondData))
}

func TestWriteFilesAtomicallyRollsBackCommittedFilesOnFailure(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.json")
	blockedPath := filepath.Join(root, "blocked")

	original := `{"name":"original"}`
	require.NoError(t, os.WriteFile(firstPath, []byte(original), 0o644))
	require.NoError(t, os.MkdirAll(blockedPath, 0o755))

	err := WriteFilesAtomically([]AtomicFileWrite{
		{
			Path:  firstPath,
			Label: "first file",
			Data:  []byte(`{"name":"updated"}`),
			Perm:  0o644,
		},
		{
			Path:  blockedPath,
			Label: "blocked file",
			Data:  []byte(`{"name":"blocked"}`),
			Perm:  0o644,
		},
	})
	require.Error(t, err)

	restored, readErr := os.ReadFile(firstPath)
	require.NoError(t, readErr)
	require.JSONEq(t, original, string(restored))
}

func TestRecoverAtomicBackupRestoresMissingTarget(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "state.json")
	backupPath := targetPath + ".bak"

	require.NoError(t, os.WriteFile(backupPath, []byte(`{"state":"backup"}`), 0o644))
	require.NoError(t, recoverAtomicBackup(targetPath, "state", false))

	recovered, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	require.JSONEq(t, `{"state":"backup"}`, string(recovered))

	_, backupErr := os.Stat(backupPath)
	require.True(t, errors.Is(backupErr, fs.ErrNotExist))
}

func TestRecoverAtomicBackupRemovesStaleBackupWhenTargetExists(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "state.json")
	backupPath := targetPath + ".bak"

	require.NoError(t, os.WriteFile(targetPath, []byte(`{"state":"current"}`), 0o644))
	require.NoError(t, os.WriteFile(backupPath, []byte(`{"state":"stale"}`), 0o644))
	require.NoError(t, recoverAtomicBackup(targetPath, "state", false))

	_, backupErr := os.Stat(backupPath)
	require.True(t, errors.Is(backupErr, fs.ErrNotExist))
}

func TestWritePathsAtomicallyReplacesDirectoryContents(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "mods", "mod-a")
	require.NoError(t, os.MkdirAll(targetPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(targetPath, "stale.txt"), []byte("stale"), 0o644))

	err := WritePathsAtomically([]AtomicWrite{
		AtomicDirectoryWrite{
			Path:  targetPath,
			Label: "mod directory",
			Callback: func(stagingPath string) error {
				return os.WriteFile(filepath.Join(stagingPath, "fresh.txt"), []byte("fresh"), 0o644)
			},
		},
	})
	require.NoError(t, err)

	_, staleErr := os.Stat(filepath.Join(targetPath, "stale.txt"))
	require.True(t, errors.Is(staleErr, fs.ErrNotExist))

	freshData, freshErr := os.ReadFile(filepath.Join(targetPath, "fresh.txt"))
	require.NoError(t, freshErr)
	require.Equal(t, "fresh", string(freshData))
}

func TestResolveManagedAtomicStagingDirUsesConfiguredRoots(t *testing.T) {
	appRoot := t.TempDir()
	metroRoot := t.TempDir()
	appStaging := filepath.Join(appRoot, "atomic-staging")
	metroStaging := filepath.Join(metroRoot, ".railyard", "atomic-staging")
	configureAtomicStagingRootsForTest(t, []StagingRoot{
		{TargetRoot: appRoot, StagingRoot: appStaging},
		{TargetRoot: metroRoot, StagingRoot: metroStaging},
	})

	appTarget := filepath.Join(appRoot, "installed_maps.json")
	metroTarget := filepath.Join(metroRoot, "cities", "data", "AAA", "roads.geojson.gz")
	otherTarget := filepath.Join(t.TempDir(), "other.json")

	require.Equal(t, filepath.Clean(appStaging), resolveManagedAtomicStagingDir(appTarget))
	require.Equal(t, filepath.Clean(metroStaging), resolveManagedAtomicStagingDir(metroTarget))
	require.Equal(t, filepath.Dir(filepath.Clean(otherTarget)), resolveManagedAtomicStagingDir(otherTarget))
}

func TestWritePathsAtomicallyUsesManagedStagingRootForDirectoryWrites(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, ".railyard", "atomic-staging")
	configureAtomicStagingRootsForTest(t, []StagingRoot{
		{TargetRoot: root, StagingRoot: stagingRoot},
	})

	targetPath := filepath.Join(root, "mods", "mod-a")
	var callbackStagingPath string
	err := WritePathsAtomically([]AtomicWrite{
		AtomicDirectoryWrite{
			Path:  targetPath,
			Label: "mod directory",
			Callback: func(stagingPath string) error {
				callbackStagingPath = stagingPath
				return os.WriteFile(filepath.Join(stagingPath, "fresh.txt"), []byte("fresh"), 0o644)
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, callbackStagingPath)
	require.True(t, pathWithinRoot(callbackStagingPath, stagingRoot))

	parentEntries, readErr := os.ReadDir(filepath.Dir(targetPath))
	require.NoError(t, readErr)
	for _, entry := range parentEntries {
		require.False(t, strings.Contains(entry.Name(), ".mod-a.tmp-"), "staging artifact should not be created in target parent")
	}
}

func TestCleanupManagedAtomicStagingRootsRemovesConfiguredRoots(t *testing.T) {
	root := t.TempDir()
	appStaging := filepath.Join(root, "app_staging")
	metroStaging := filepath.Join(root, "metro_staging")
	configureAtomicStagingRootsForTest(t, []StagingRoot{
		{TargetRoot: filepath.Join(root, "app"), StagingRoot: appStaging},
		{TargetRoot: filepath.Join(root, "metro"), StagingRoot: metroStaging},
	})

	require.NoError(t, os.MkdirAll(filepath.Join(appStaging, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appStaging, "nested", "stale.tmp"), []byte("a"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(metroStaging, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(metroStaging, "nested", "stale.tmp"), []byte("b"), 0o644))

	require.NoError(t, CleanupTmpStagingRoots())
	_, appErr := os.Stat(appStaging)
	require.ErrorIs(t, appErr, os.ErrNotExist)
	_, metroErr := os.Stat(metroStaging)
	require.ErrorIs(t, metroErr, os.ErrNotExist)
}

func TestWritePathsAtomicallyKeepsOriginalDirectoryWhenPopulateFails(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "maps", "AAA")
	require.NoError(t, os.MkdirAll(targetPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(targetPath, "existing.txt"), []byte("keep"), 0o644))

	err := WritePathsAtomically([]AtomicWrite{
		AtomicDirectoryWrite{
			Path:  targetPath,
			Label: "map directory",
			Callback: func(string) error {
				return errors.New("populate failed")
			},
		},
	})
	require.Error(t, err)

	existingData, existingErr := os.ReadFile(filepath.Join(targetPath, "existing.txt"))
	require.NoError(t, existingErr)
	require.Equal(t, "keep", string(existingData))
}

func TestWritePathsAtomicallyMixedWritesRollBackOnFailure(t *testing.T) {
	root := t.TempDir()
	dirPath := filepath.Join(root, "cities", "data", "AAA")
	blockedPath := filepath.Join(root, "tiles", "AAA.pmtiles")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirPath, "existing.gz"), []byte("old"), 0o644))
	require.NoError(t, os.MkdirAll(blockedPath, 0o755))

	err := WritePathsAtomically([]AtomicWrite{
		AtomicDirectoryWrite{
			Path:  dirPath,
			Label: "map data directory",
			Callback: func(stagingPath string) error {
				return os.WriteFile(filepath.Join(stagingPath, "fresh.gz"), []byte("new"), 0o644)
			},
		},
		AtomicFileWrite{
			Path:  blockedPath,
			Label: "map tiles",
			Data:  []byte("tiles"),
		},
	})
	require.Error(t, err)

	_, oldErr := os.Stat(filepath.Join(dirPath, "existing.gz"))
	require.NoError(t, oldErr)
	_, freshErr := os.Stat(filepath.Join(dirPath, "fresh.gz"))
	require.ErrorIs(t, freshErr, os.ErrNotExist)
}

func TestWritePathsAtomicallySupportsStagedFileWrites(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "thumbs", "AAA.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	stagedFile, err := os.CreateTemp(filepath.Dir(targetPath), ".AAA.png.tmp-*")
	require.NoError(t, err)
	_, err = stagedFile.Write([]byte("png-bytes"))
	require.NoError(t, err)
	require.NoError(t, stagedFile.Close())

	err = WritePathsAtomically([]AtomicWrite{
		AtomicFileWrite{
			Path:       targetPath,
			Label:      "map thumbnail",
			StagedPath: stagedFile.Name(),
		},
	})
	require.NoError(t, err)

	data, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	require.Equal(t, "png-bytes", string(data))
}
