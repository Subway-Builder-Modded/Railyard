package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/logger"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestCopyDirFromFS(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "nested", "data.txt"), []byte("payload"), 0o644))

	dest := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, CopyDirFromFS(dest, os.DirFS(source)))

	bytes, err := os.ReadFile(filepath.Join(dest, "nested", "data.txt"))
	require.NoError(t, err)
	require.Equal(t, "payload", string(bytes))
}

func TestCopyDirFromFSOverwritesExistingFiles(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "nested", "data.txt"), []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(source, ".railyard_asset"), []byte("marker"), 0o644))

	dest := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "nested", "data.txt"), []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dest, ".railyard_asset"), []byte("old-marker"), 0o644))

	require.NoError(t, CopyDirFromFS(dest, os.DirFS(source)))

	bytes, err := os.ReadFile(filepath.Join(dest, "nested", "data.txt"))
	require.NoError(t, err)
	require.Equal(t, "new", string(bytes))

	marker, err := os.ReadFile(filepath.Join(dest, ".railyard_asset"))
	require.NoError(t, err)
	require.Equal(t, "marker", string(marker))
}

func TestCopyOptionalFile(t *testing.T) {
	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "optional.log"))
	source := filepath.Join(t.TempDir(), "source.txt")
	dest := filepath.Join(t.TempDir(), "dest", "target.txt")

	response, ok := CopyOptionalFile(source, dest, "profile-a", "map-a", "thumbnail", log)
	require.True(t, ok)
	require.Equal(t, types.GenericResponse{}, response)
	_, statErr := os.Stat(dest)
	require.True(t, errors.Is(statErr, fs.ErrNotExist))

	require.NoError(t, os.WriteFile(source, []byte("file-data"), 0o644))
	response, ok = CopyOptionalFile(source, dest, "profile-a", "map-a", "thumbnail", log)
	require.True(t, ok)
	require.Equal(t, types.GenericResponse{}, response)
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "file-data", string(data))
}
