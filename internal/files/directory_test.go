package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagedDirectorySize(t *testing.T) {
	t.Run("returns zero when directory is missing", func(t *testing.T) {
		size, err := ManagedDirectorySize(filepath.Join(t.TempDir(), "missing"), ".marker")
		require.NoError(t, err)
		require.Equal(t, int64(0), size)
	})

	t.Run("returns zero when marker is missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte("1234"), 0o644))
		size, err := ManagedDirectorySize(dir, ".marker")
		require.NoError(t, err)
		require.Equal(t, int64(0), size)
	})

	t.Run("returns size when marker exists", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".marker"), []byte("m"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.bin"), []byte("12345"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "nested"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "b.bin"), []byte("678"), 0o644))

		size, err := ManagedDirectorySize(dir, ".marker")
		require.NoError(t, err)
		// marker (1) + a.bin (5) + b.bin (3)
		require.Equal(t, int64(9), size)
	})
}

func TestHasAssetMarker(t *testing.T) {
	t.Run("returns false when directory is missing", func(t *testing.T) {
		found, err := HasAssetMarker(filepath.Join(t.TempDir(), "missing"), ".marker")
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("returns false when marker is missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte("1234"), 0o644))
		found, err := HasAssetMarker(dir, ".marker")
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("returns true when marker exists", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".marker"), []byte("m"), 0o644))
		found, err := HasAssetMarker(dir, ".marker")
		require.NoError(t, err)
		require.True(t, found)
	})
}
