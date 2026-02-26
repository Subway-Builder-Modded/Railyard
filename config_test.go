package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setEnv(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("APPDATA", root)         // Config directory for Windows
	t.Setenv("XDG_CONFIG_HOME", root) // Config directory for Linux and MacOS
	t.Setenv("HOME", root)            // Fallback for non-windows OS
}

func writeTestConfigFile(t *testing.T, content string) {
	t.Helper()

	path := ConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func testConfig() AppConfig {
	return AppConfig{
		ExecutablePath: "dir/executable.exe",
		ModFolderPath:  "dir/mods",
	}
}

func TestAreConfigPathsConfigured(t *testing.T) {
	cfg := testConfig()
	require.True(t, cfg.areConfigPathsConfigured())

	cfg.ModFolderPath = "   "
	require.False(t, cfg.areConfigPathsConfigured())
}

func TestValidateConfigPaths(t *testing.T) {
	setEnv(t)

	// Paths not configured
	cfg := AppConfig{}
	valid, result := cfg.ValidateConfigPaths()
	require.False(t, valid)
	require.False(t, result.IsConfigured)

	// Paths are configured but do not exist on disk
	cfg = AppConfig{
		ModFolderPath:  "blah/blah/mods",
		ExecutablePath: "blah.exe",
	}
	valid, result = cfg.ValidateConfigPaths()
	require.False(t, valid)
	require.True(t, result.IsConfigured)
	require.False(t, result.ModFolderPathValid)
	require.False(t, result.ExecutablePathValid)

	modDir := t.TempDir()
	exeFile := filepath.Join(modDir, "abcdef.exe")
	require.NoError(t, os.WriteFile(exeFile, []byte(""), 0o644))

	// Paths are configured and exist on disk
	cfg = AppConfig{
		ModFolderPath:  modDir,
		ExecutablePath: exeFile,
	}
	valid, result = cfg.ValidateConfigPaths()
	require.True(t, valid)
	require.True(t, result.IsConfigured)
	require.True(t, result.ModFolderPathValid)
	require.True(t, result.ExecutablePathValid)
}

func TestUpdateConfigPersistsMutations(t *testing.T) {
	setEnv(t)
	require.NoError(t, writeAppConfig(AppConfig{
		ExecutablePath: "dir/executable.exe",
	}))

	cfg := NewConfig()
	updated, err := cfg.updateConfig(func(c *AppConfig) {
		c.ModFolderPath = "dir/mods"
	})
	require.NoError(t, err)
	require.Equal(t, testConfig(), updated)

	persisted, err := readAppConfig()
	require.NoError(t, err)
	require.Equal(t, updated, persisted)
}

func TestSetConfigOverwritesAllFields(t *testing.T) {
	setEnv(t)
	require.NoError(t, writeAppConfig(testConfig()))

	cfg := NewConfig()
	next := AppConfig{
		ExecutablePath: "new/executable.exe",
		ModFolderPath:  "new/mods",
	}
	updated, err := cfg.SetConfig(next)
	require.NoError(t, err)
	require.Equal(t, next, updated)

	// updated config should be persisted to disk
	persisted, err := readAppConfig()
	require.NoError(t, err)
	require.Equal(t, next, persisted)
}

func TestClearConfigOverwritesWithEmptyConfig(t *testing.T) {
	setEnv(t)
	require.NoError(t, writeAppConfig(testConfig()))

	cfg := NewConfig()
	updated, err := cfg.ClearConfig()
	require.NoError(t, err)
	require.Equal(t, AppConfig{}, updated)

	persisted, err := readAppConfig()
	require.NoError(t, err)
	require.Equal(t, AppConfig{}, persisted)
}
