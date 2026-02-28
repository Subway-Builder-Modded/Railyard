package main

import (
	"os"
	"path/filepath"
	"railyard/internal/types"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestSetup struct {
	t   *testing.T
	cfg *Config
}

func setEnv(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("APPDATA", root)           // Config directory for Windows
	t.Setenv("LOCALAPPDATA", root)      // Executable default candidate on Windows
	t.Setenv("ProgramFiles", root)      // Executable default candidate fallback on Windows
	t.Setenv("ProgramFiles(x86)", root) // Executable default candidate fallback on Windows
	t.Setenv("XDG_CONFIG_HOME", root)   // Config directory for Linux and MacOS
	t.Setenv("HOME", root)              // Fallback for non-windows OS
}

func tryResolveConfig(t *testing.T, cfg *Config) {
	t.Helper()
	_, err := cfg.resolveConfig()
	require.NoError(t, err)
}

func setup(t *testing.T, persisted types.AppConfig) *TestSetup {
	t.Helper()
	setEnv(t)
	require.NoError(t, writeAppConfig(persisted))

	c := NewConfig()
	tryResolveConfig(t, c)

	return &TestSetup{t: t, cfg: c}
}

func (h *TestSetup) persisted() types.AppConfig {
	h.t.Helper()
	persisted, err := readAppConfig()
	require.NoError(h.t, err)
	return persisted
}

func (h *TestSetup) runtime() types.ResolveConfigResult {
	h.t.Helper()
	return h.cfg.GetConfig()
}

func testConfig() types.AppConfig {
	return types.AppConfig{
		ExecutablePath:     "dir/executable.exe",
		MetroMakerDataPath: "dir/",
	}
}

func testCandidatePaths(t *testing.T) []string {
	root := t.TempDir()

	filePath := filepath.Join(root, "candidate.exe")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))
	dirPath := filepath.Join(root, "metro-maker4")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	return []string{
		"",
		"relative/path",
		filePath,
		dirPath,
	}
}

func TestUpdateConfigWithPersist(t *testing.T) {
	h := setup(t, types.AppConfig{
		ExecutablePath: "dir/executable.exe",
	})

	updated, err := h.cfg.updateConfig(func(c *types.AppConfig) {
		c.MetroMakerDataPath = "dir/"
	}, true) // Write through to disk

	require.NoError(t, err)
	require.Equal(t, testConfig(), updated.Config)
	require.Equal(t, updated.Config, h.persisted())
}

func TestUpdateConfigWithoutPersist(t *testing.T) {
	original := testConfig()
	h := setup(t, original)

	updated, err := h.cfg.updateConfig(func(c *types.AppConfig) {
		c.ExecutablePath = "updated/executable.exe"
	}, false)
	require.NoError(t, err)
	// cfg in memory should be updated; and cfg in the result from updateConfig should point to the same object
	require.Equal(t, "updated/executable.exe", updated.Config.ExecutablePath)
	require.Equal(t, "updated/executable.exe", h.runtime().Config.ExecutablePath)

	require.Equal(t, original, h.persisted())
}

func TestSaveConfigPersistsRuntimeState(t *testing.T) {
	h := setup(t, types.AppConfig{})

	updated, err := h.cfg.updateConfig(func(c *types.AppConfig) {
		c.MetroMakerDataPath = "runtime/metro-maker4"
		c.ExecutablePath = "runtime/Subway Builder.exe"
	}, false)
	require.NoError(t, err)
	require.Equal(t, "runtime/metro-maker4", updated.Config.MetroMakerDataPath)

	saved, err := h.cfg.SaveConfig()
	require.NoError(t, err)
	require.Equal(t, updated.Config, saved.Config)
	require.Equal(t, saved.Config, h.persisted())
}

func TestResolveConfigOverridesRuntimeState(t *testing.T) {
	setEnv(t)
	initial := types.AppConfig{
		MetroMakerDataPath: "first/metro",
		ExecutablePath:     "first.exe",
	}
	updated := types.AppConfig{
		MetroMakerDataPath: "second/metro",
		ExecutablePath:     "second.exe",
	}

	require.NoError(t, writeAppConfig(initial))
	cfg := NewConfig()

	resolved, err := cfg.resolveConfig()
	require.NoError(t, err)
	require.Equal(t, initial, resolved.Config)

	require.NoError(t, writeAppConfig(updated))
	runtimeBeforeReload := cfg.GetConfig()
	require.Equal(t, initial, runtimeBeforeReload.Config)

	reloaded, err := cfg.resolveConfig()
	require.NoError(t, err)
	require.Equal(t, updated, reloaded.Config)
}

func TestSetConfigOverwritesRuntime(t *testing.T) {
	original := testConfig()
	h := setup(t, original)

	next := types.AppConfig{
		ExecutablePath:     "new/executable.exe",
		MetroMakerDataPath: "new/",
	}
	updated, err := h.cfg.SetConfig(next)
	require.NoError(t, err)
	require.Equal(t, next, updated)

	runtimeConfig := h.runtime()
	require.Equal(t, next, runtimeConfig.Config)

	// SetConfig should only affect the runtime config; no mutation should occur to the persisted config
	require.Equal(t, original, h.persisted())
}

func TestClearConfigOverwritesRuntimeWithEmptyConfig(t *testing.T) {
	original := testConfig()
	h := setup(t, original)

	updated, err := h.cfg.ClearConfig()
	require.NoError(t, err)
	require.Equal(t, types.AppConfig{}, updated)

	runtimeConfig := h.runtime()
	require.Equal(t, types.AppConfig{}, runtimeConfig.Config)

	// ClearConfig should only affect the runtime config; no mutation should occur to the persisted config
	require.Equal(t, original, h.persisted())
}

func TestFindDefaultPathReturnsFirstMatchingDirectory(t *testing.T) {
	candidatePaths := testCandidatePaths(t)
	found, success := findDefaultPath(candidatePaths, true)
	require.True(t, success)
	require.Equal(t, candidatePaths[3], found)
}

func TestFindDefaultPathReturnsFirstMatchingFile(t *testing.T) {
	candidatePaths := testCandidatePaths(t)
	found, success := findDefaultPath(candidatePaths, false)
	require.True(t, success)
	require.Equal(t, candidatePaths[2], found)
}

func TestFindDefaultPathReturnsNotFoundWhenTypeMismatches(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "candidate.exe")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	// Executable file does not match when looking for directory
	found, success := findDefaultPath([]string{filePath}, true)
	require.False(t, success)
	require.Equal(t, "", found)
}

func createWritableCandidateFile(t *testing.T, candidates []string) string {
	t.Helper()

	candidate, success := firstValidCandidate(candidates)
	if !success {
		t.Skip("no valid default executable candidate path available")
		return ""
	}

	require.NoError(t, os.MkdirAll(filepath.Dir(candidate), 0o755))
	require.NoError(t, os.WriteFile(candidate, []byte("x"), 0o755))
	return candidate
}

func createWritableCandidateDir(t *testing.T, candidates []string) string {
	t.Helper()

	candidate, success := firstValidCandidate(candidates)
	if !success {
		t.Skip("no valid default metro maker data folder candidate path available")
		return ""
	}

	require.NoError(t, os.MkdirAll(candidate, 0o755))
	return candidate
}

func firstValidCandidate(candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if candidate != "" && filepath.IsAbs(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func TestOpenExecutableDialogAutoDetectSuccessDoesNotPersist(t *testing.T) {
	h := setup(t, types.AppConfig{})
	metroMakerPath := t.TempDir()

	_, err := h.cfg.updateMetroMakerDataFolder(metroMakerPath)
	require.NoError(t, err)
	_, err = h.cfg.SaveConfig()
	require.NoError(t, err)
	detectedPath := createWritableCandidateFile(t, defaultExecutableCandidates())

	result, err := h.cfg.OpenExecutableDialog(types.SetConfigPathOptions{AllowAutoDetect: true})
	require.NoError(t, err)
	require.Equal(t, types.SourceAutoDetected, result.SetConfigSource)
	require.Equal(t, detectedPath, result.AutoDetectedPath)
	require.Equal(t, detectedPath, result.ResolveConfigResult.Config.ExecutablePath)

	runtimeCfg := h.runtime()
	require.Equal(t, detectedPath, runtimeCfg.Config.ExecutablePath)

	require.Equal(t, types.AppConfig{
		MetroMakerDataPath: metroMakerPath,
	}, h.persisted())
}

func TestOpenMetroMakerDialogAutoDetectSuccessDoesNotPersist(t *testing.T) {
	h := setup(t, types.AppConfig{})
	executablePath := createWritableCandidateFile(t, defaultExecutableCandidates())

	_, err := h.cfg.updateExecutable(executablePath)
	require.NoError(t, err)
	_, err = h.cfg.SaveConfig()
	require.NoError(t, err)
	detectedPath := createWritableCandidateDir(t, defaultMetroMakerDataFolderCandidates())

	result, err := h.cfg.OpenMetroMakerDataFolderDialog(types.SetConfigPathOptions{AllowAutoDetect: true})
	require.NoError(t, err)
	require.Equal(t, types.SourceAutoDetected, result.SetConfigSource)
	require.Equal(t, detectedPath, result.AutoDetectedPath)
	require.Equal(t, detectedPath, result.ResolveConfigResult.Config.MetroMakerDataPath)

	runtimeCfg := h.runtime()
	require.Equal(t, detectedPath, runtimeCfg.Config.MetroMakerDataPath)

	require.Equal(t, types.AppConfig{
		ExecutablePath: executablePath,
	}, h.persisted())
}

func TestTryAutoDetectExecutableFailsValidation(t *testing.T) {
	setEnv(t)
	detectedPath := createWritableCandidateFile(t, defaultExecutableCandidates())

	cfg := NewConfig()
	autoDetected, success := cfg.tryAutoDetectPath(
		defaultExecutableCandidates(),
		false,
		cfg.updateExecutable,
	)
	require.False(t, success)
	require.Equal(t, types.SetConfigPathResult{}, autoDetected)

	runtimeAfter := cfg.GetConfig()
	require.Equal(t, detectedPath, runtimeAfter.Config.ExecutablePath)
}

func TestTryAutoDetectMetroMakerFailsValidation(t *testing.T) {
	setEnv(t)
	detectedPath := createWritableCandidateDir(t, defaultMetroMakerDataFolderCandidates())

	cfg := NewConfig()
	autoDetected, success := cfg.tryAutoDetectPath(
		defaultMetroMakerDataFolderCandidates(),
		true,
		cfg.updateMetroMakerDataFolder,
	)
	require.False(t, success)
	require.Equal(t, types.SetConfigPathResult{}, autoDetected)

	runtimeAfter := cfg.GetConfig()
	require.Equal(t, detectedPath, runtimeAfter.Config.MetroMakerDataPath)
}
