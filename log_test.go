package main

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppLoggerShutdownFlushesPendingBuffer(t *testing.T) {
	setEnv(t)

	logger := NewAppLogger()
	require.NoError(t, logger.Start())

	logger.SetRoutine(RuntimeRoutine)
	logger.logger().Info("flush on shutdown")
	require.NoError(t, logger.Shutdown())

	data, err := os.ReadFile(LogFilePath())
	require.NoError(t, err)
	require.Contains(t, string(data), "flush on shutdown")
}

func TestAppLoggerErrorIncludesErrorField(t *testing.T) {
	setEnv(t)

	logger := NewAppLogger()
	require.NoError(t, logger.Start())

	logger.SetRoutine(RuntimeRoutine)
	logger.Error("operation failed", errors.New("boom"), "component", "test")
	require.NoError(t, logger.Shutdown())

	data, err := os.ReadFile(LogFilePath())
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "level=ERROR")
	require.Contains(t, content, "operation failed")
	require.Contains(t, content, "error=boom")
	require.Contains(t, content, "component=test")
}

func TestAppLoggerShutdownIsIdempotent(t *testing.T) {
	setEnv(t)

	logger := NewAppLogger()
	require.NoError(t, logger.Start())
	logger.logger().Info("first")
	require.NoError(t, logger.Shutdown())
	require.NoError(t, logger.Shutdown())
}
