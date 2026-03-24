package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	require.NoError(t, recoverAtomicBackup(targetPath, "state"))

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
	require.NoError(t, recoverAtomicBackup(targetPath, "state"))

	_, backupErr := os.Stat(backupPath)
	require.True(t, errors.Is(backupErr, fs.ErrNotExist))
}
