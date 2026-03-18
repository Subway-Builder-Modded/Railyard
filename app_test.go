package main

import (
	"testing"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestIsStartupReady(t *testing.T) {
	app := &App{}
	require.False(t, app.IsStartupReady().Ready)

	app.setStartupReady(true)
	require.True(t, app.IsStartupReady().Ready)

	app.setStartupReady(false)
	require.False(t, app.IsStartupReady().Ready)
}

func TestOpenInFileExplorerRejectsInvalidPaths(t *testing.T) {
	app := &App{}

	empty := app.OpenInFileExplorer("   ")
	require.Equal(t, types.ResponseError, empty.Status)
	require.Equal(t, "invalid path", empty.Message)

	missing := app.OpenInFileExplorer("this-path-does-not-exist")
	require.Equal(t, types.ResponseError, missing.Status)
	require.Contains(t, missing.Message, "failed to resolve path")
}
