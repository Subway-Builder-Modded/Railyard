package paths

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type quarantineLogger struct {
	warned  bool
	errored bool
}

func (l *quarantineLogger) Error(_ string, _ error, _ ...any) {
	l.errored = true
}

func (l *quarantineLogger) Warn(_ string, _ ...any) {
	l.warned = true
}

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

func TestNormalizeLocalPath(t *testing.T) {
	require.Equal(t, "", NormalizeLocalPath("   "))

	got := NormalizeLocalPath(`metro-maker4\cities/data\DCA`)
	if runtime.GOOS == "windows" {
		require.NotContains(t, got, "/")
		require.Contains(t, got, `\`)
	} else {
		require.NotContains(t, got, `\`)
		require.Contains(t, got, "/")
	}
	require.True(t, strings.HasSuffix(got, filepath.Join("cities", "data", "DCA")))
}

func TestJoinLocalPath(t *testing.T) {
	base := filepath.Join("Users", "alex", "metro-maker4")
	// Simulate legacy mixed separator input.
	base = strings.ReplaceAll(base, string(filepath.Separator), `\`)

	got := JoinLocalPath(base, "cities", "data", "DCA")
	if runtime.GOOS != "windows" {
		require.NotContains(t, got, `\`)
	}
	require.True(t, strings.HasSuffix(got, filepath.Join("cities", "data", "DCA")))
}

func TestQuarantineFile(t *testing.T) {
	setEnv(t)

	t.Run("Missing source is no-op", func(t *testing.T) {
		logger := &quarantineLogger{}
		success, backupPath := QuarantineFile(filepath.Join(t.TempDir(), "missing.json"), logger)
		require.True(t, success)
		require.Empty(t, backupPath)
		require.False(t, logger.warned)
		require.False(t, logger.errored)
	})

	t.Run("Moves file and logs warning", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "user_profiles.json")
		require.NoError(t, os.WriteFile(src, []byte(`{}`), 0o644))

		logger := &quarantineLogger{}
		success, backupPath := QuarantineFile(src, logger)
		require.True(t, success)
		require.NotEmpty(t, backupPath)
		require.True(t, logger.warned)
		require.False(t, logger.errored)
		_, err := os.Stat(src)
		require.True(t, errors.Is(err, fs.ErrNotExist))
		_, err = os.Stat(backupPath)
		require.NoError(t, err)
	})
}

func TestMoveLogFile(t *testing.T) {
	setEnv(t)

	t.Run("No current log file", func(t *testing.T) {
		require.NoError(t, MoveLogFile())
	})

	t.Run("Rotates current log to previous log path", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(AppDataRoot(), 0o755))
		require.NoError(t, os.WriteFile(LogFilePath(), []byte("line"), 0o644))
		require.NoError(t, MoveLogFile())

		_, err := os.Stat(LogFilePath())
		require.True(t, errors.Is(err, fs.ErrNotExist))

		data, err := os.ReadFile(PrevLogFilePath())
		require.NoError(t, err)
		require.Equal(t, "line", string(data))
	})
}
