package logger

import (
	"errors"
	"os"
	"path/filepath"
	"railyard/internal/paths"
	"railyard/internal/types"
	"testing"

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

func readLogContent(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(paths.LogFilePath())
	require.NoError(t, err)
	return string(data)
}

func TestAppLoggerStartIsIdempotent(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Start())
	require.NoError(t, l.Start())
	require.NoError(t, l.Shutdown())
}

func TestAppLoggerShutdownBeforeStartIsNoOp(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Shutdown())
}

func TestAppLoggerShutdownIsIdempotent(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Start())
	l.Info("MEOW")
	require.NoError(t, l.Shutdown())
	require.NoError(t, l.Shutdown())
}

func TestAppLoggerWritesBeforeStartAreDropped(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	l.Info("no meow :(")
	require.NoError(t, l.Start())
	require.NoError(t, l.Shutdown())

	content := readLogContent(t)
	require.NotContains(t, content, "no meow :(")
}

func TestAppLoggerShutdownFlushesBuffer(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Start())

	l.Info("meow remains")
	require.NoError(t, l.Shutdown())

	content := readLogContent(t)
	require.Contains(t, content, "meow remains")
}

func TestAppLoggerErrorIncludesErrorField(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Start())

	l.Error("cat invasion", errors.New("meow"), "kitty", "bad")
	require.NoError(t, l.Shutdown())

	content := readLogContent(t)
	require.Contains(t, content, "level=ERROR")
	require.Contains(t, content, "cat invasion")
	require.Contains(t, content, "error=meow")
	require.Contains(t, content, "kitty=bad")
}

func TestAppLoggerCanRestartAfterShutdown(t *testing.T) {
	setEnv(t)

	l := NewAppLogger()
	require.NoError(t, l.Start())
	l.Info("first meow")
	require.NoError(t, l.Shutdown())

	require.NoError(t, l.Start())
	l.Info("second meow")
	require.NoError(t, l.Shutdown())

	content := readLogContent(t)
	require.Contains(t, content, "first meow")
	require.Contains(t, content, "second meow")
}

func TestAppLoggerLogResponseMapsStatusToLevels(t *testing.T) {
	setEnv(t)

	l := LoggerAtPath(filepath.Join(t.TempDir(), "test.log"))
	require.NoError(t, l.Start())

	l.LogResponse("ok response", types.GenericResponse{
		Status:  types.ResponseSuccess,
		Message: "all good",
	})
	l.LogResponse("warn response", types.GenericResponse{
		Status:  types.ResponseWarn,
		Message: "heads up",
	})
	l.LogResponse("error response", types.GenericResponse{
		Status:  types.ResponseError,
		Message: "something failed",
	})
	require.NoError(t, l.Shutdown())

	data, err := os.ReadFile(l.path)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "level=INFO")
	require.Contains(t, content, "level=WARN")
	require.Contains(t, content, "level=ERROR")
	require.Contains(t, content, "status=success")
	require.Contains(t, content, "status=warn")
	require.Contains(t, content, "status=error")
}
