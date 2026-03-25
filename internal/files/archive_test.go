package files

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"railyard/internal/logger"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

type archiveEntryStub struct {
	data []byte
}

func (s archiveEntryStub) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.data)), nil
}

func TestCopyFileToArchiveAndExtractArchiveToDir(t *testing.T) {
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "sample.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "single.tar")
	rawArchive, err := os.Create(archivePath)
	require.NoError(t, err)

	writer := tar.NewWriter(rawArchive)
	require.NoError(t, CopyFileToArchive(writer, sourceFile))
	require.NoError(t, writer.Close())
	require.NoError(t, rawArchive.Close())

	extractDir := t.TempDir()
	require.NoError(t, ExtractArchiveToDir(archivePath, extractDir))

	data, readErr := os.ReadFile(filepath.Join(extractDir, "sample.txt"))
	require.NoError(t, readErr)
	require.Equal(t, "hello", string(data))
}

func TestAddDirToArchiveAndExtractArchiveToDir(t *testing.T) {
	sourceDir := t.TempDir()
	nested := filepath.Join(sourceDir, "nested", "child.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(nested), 0o755))
	require.NoError(t, os.WriteFile(nested, []byte("nested-data"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "dir.tar")
	rawArchive, err := os.Create(archivePath)
	require.NoError(t, err)

	writer := tar.NewWriter(rawArchive)
	require.NoError(t, AddDirToArchive(writer, sourceDir, sourceDir))
	require.NoError(t, writer.Close())
	require.NoError(t, rawArchive.Close())

	extractDir := t.TempDir()
	require.NoError(t, ExtractArchiveToDir(archivePath, extractDir))

	data, readErr := os.ReadFile(filepath.Join(extractDir, "nested", "child.txt"))
	require.NoError(t, readErr)
	require.Equal(t, "nested-data", string(data))
}

func TestCopyDirectory(t *testing.T) {
	sourceDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "a", "b"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "a", "b", "file.txt"), []byte("copy-me"), 0o644))

	destination := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, CopyDirectory(sourceDir, destination))

	data, err := os.ReadFile(filepath.Join(destination, "a", "b", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "copy-me", string(data))
}

func TestCopyFileWithDest(t *testing.T) {
	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "archive_test.log"))

	src := filepath.Join(t.TempDir(), "from.txt")
	require.NoError(t, os.WriteFile(src, []byte("payload"), 0o644))

	dst := filepath.Join(t.TempDir(), "dest", "to.txt")
	response, ok := CopyFileWithDest(src, dst, "profile-a", "map-a", "manifest", log)
	require.True(t, ok)
	require.Equal(t, types.GenericResponse{}, response)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "payload", string(data))
}

func TestCopyFileWithDestReturnsErrorForMissingSource(t *testing.T) {
	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "archive_test_error.log"))
	response, ok := CopyFileWithDest("missing", filepath.Join(t.TempDir(), "dest", "to.txt"), "profile-a", "map-a", "manifest", log)
	require.False(t, ok)
	require.Equal(t, types.ResponseError, response.Status)
}

func TestCopyFile(t *testing.T) {
	log := logger.LoggerAtPath(filepath.Join(t.TempDir(), "copy_file.log"))
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("abc"), 0o644))

	response, ok := CopyFile(src, dst, "profile-a", "map-a", log)
	require.True(t, ok)
	require.Equal(t, types.GenericResponse{}, response)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "abc", string(data))
}

func TestReadJSONFromTarArchive(t *testing.T) {
	sourceDir := t.TempDir()
	payloadPath := filepath.Join(sourceDir, "meta", "profile_subscriptions.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(payloadPath), 0o755))

	expected := types.Subscriptions{
		Maps:      map[string]string{"map-a": "1.0.0"},
		LocalMaps: map[string]string{"ABC": "0.0.0"},
		Mods:      map[string]string{"mod-a": "2.0.0"},
	}
	raw, err := json.Marshal(expected)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(payloadPath, raw, 0o644))

	archivePath := filepath.Join(t.TempDir(), "subs.tar")
	rawArchive, err := os.Create(archivePath)
	require.NoError(t, err)

	writer := tar.NewWriter(rawArchive)
	require.NoError(t, AddDirToArchive(writer, sourceDir, sourceDir))
	require.NoError(t, writer.Close())
	require.NoError(t, rawArchive.Close())

	actual, found, readErr := ReadJSONFromTarArchive[types.Subscriptions](archivePath, "profile_subscriptions.json")
	require.NoError(t, readErr)
	require.True(t, found)
	require.Equal(t, expected, actual)
}

func TestReadJSONFromTarArchiveReturnsNotFound(t *testing.T) {
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "sample.txt"), []byte("hello"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "missing-json.tar")
	rawArchive, err := os.Create(archivePath)
	require.NoError(t, err)
	writer := tar.NewWriter(rawArchive)
	require.NoError(t, AddDirToArchive(writer, sourceDir, sourceDir))
	require.NoError(t, writer.Close())
	require.NoError(t, rawArchive.Close())

	actual, found, readErr := ReadJSONFromTarArchive[types.Subscriptions](archivePath, "profile_subscriptions.json")
	require.NoError(t, readErr)
	require.False(t, found)
	require.Equal(t, types.Subscriptions{}, actual)
}

func TestReadJSONFromTarArchiveReturnsDecodeError(t *testing.T) {
	sourceDir := t.TempDir()
	payloadPath := filepath.Join(sourceDir, "meta", "profile_subscriptions.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(payloadPath), 0o755))
	require.NoError(t, os.WriteFile(payloadPath, []byte("{invalid"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "invalid-json.tar")
	rawArchive, err := os.Create(archivePath)
	require.NoError(t, err)
	writer := tar.NewWriter(rawArchive)
	require.NoError(t, AddDirToArchive(writer, sourceDir, sourceDir))
	require.NoError(t, writer.Close())
	require.NoError(t, rawArchive.Close())

	_, found, readErr := ReadJSONFromTarArchive[types.Subscriptions](archivePath, "profile_subscriptions.json")
	require.Error(t, readErr)
	require.False(t, found)
}

func TestWriteArchiveJSON(t *testing.T) {
	tempDir := t.TempDir()
	payload := map[string]string{"map-a": "1.0.0"}
	require.NoError(t, WriteArchiveJSON(tempDir, "profile_subscriptions.json", "profile subscriptions", payload))

	actual, err := ReadJSON[map[string]string](
		filepath.Join(tempDir, "profile_subscriptions.json"),
		"profile subscriptions",
		JSONReadOptions{},
	)
	require.NoError(t, err)
	require.Equal(t, payload, actual)
}

func TestStageArchiveForAtomicWriteUsesManagedStagingRoot(t *testing.T) {
	root := t.TempDir()
	targetRoot := filepath.Join(root, "metro")
	stagingRoot := filepath.Join(root, ".railyard", "atomic-staging")
	configureAtomicStagingRootsForTest(t, []StagingRoot{
		{
			TargetRoot:  targetRoot,
			StagingRoot: stagingRoot,
		},
	})

	destinationPath := filepath.Join(targetRoot, "public", "data", "city-maps", "AAA.svg")
	stagedPath, err := StageArchiveForAtomicWrite(destinationPath, archiveEntryStub{data: []byte("svg-data")}, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(stagedPath) })

	require.True(t, pathWithinRoot(stagedPath, stagingRoot))
	data, readErr := os.ReadFile(stagedPath)
	require.NoError(t, readErr)
	require.Equal(t, "svg-data", string(data))
}
