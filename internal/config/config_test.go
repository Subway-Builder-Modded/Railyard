package config

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"railyard/internal/testutil"
	"railyard/internal/types"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestSetup struct {
	t   *testing.T
	cfg *Config
}

func tryResolveConfig(t *testing.T, cfg *Config) {
	t.Helper()
	_, err := cfg.ResolveConfig()
	require.NoError(t, err)
}

func setup(t *testing.T, persisted types.AppConfig) *TestSetup {
	t.Helper()
	testutil.NewHarness(t)
	require.NoError(t, WriteAppConfig(persisted))

	c := NewConfig()
	tryResolveConfig(t, c)

	return &TestSetup{t: t, cfg: c}
}

func (h *TestSetup) persisted() types.AppConfig {
	h.t.Helper()
	persisted, err := ReadAppConfig()
	require.NoError(h.t, err)
	return persisted
}

func (h *TestSetup) runtime() types.ResolveConfigResponse {
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

	updated, err := h.cfg.UpdateConfig(func(c *types.AppConfig) {
		c.MetroMakerDataPath = "dir/"
	}, true) // Write through to disk

	require.NoError(t, err)
	require.Equal(t, testConfig(), updated.Config)
	require.False(t, updated.HasGithubToken)
	require.Equal(t, updated.Config, h.persisted())
}

func TestUpdateConfigWithoutPersist(t *testing.T) {
	original := testConfig()
	h := setup(t, original)

	updated, err := h.cfg.UpdateConfig(func(c *types.AppConfig) {
		c.ExecutablePath = "updated/executable.exe"
	}, false)
	require.NoError(t, err)
	// cfg in memory should be updated; and cfg in the result from UpdateConfig should point to the same object
	require.Equal(t, "updated/executable.exe", updated.Config.ExecutablePath)
	require.Equal(t, "updated/executable.exe", h.runtime().Config.ExecutablePath)

	require.Equal(t, original, h.persisted())
}

func TestSaveConfigPersistsRuntimeState(t *testing.T) {
	h := setup(t, types.AppConfig{})

	updated, err := h.cfg.UpdateConfig(func(c *types.AppConfig) {
		c.MetroMakerDataPath = "runtime/metro-maker4"
		c.ExecutablePath = "runtime/Subway Builder.exe"
	}, false)
	require.NoError(t, err)
	require.Equal(t, "runtime/metro-maker4", updated.Config.MetroMakerDataPath)

	saved := h.cfg.SaveConfig()
	require.Equal(t, types.ResponseSuccess, saved.Status)
	require.Equal(t, updated.Config, saved.Config)
	require.Equal(t, saved.Config, h.persisted())
}

func TestResolveConfigOverridesRuntimeState(t *testing.T) {
	testutil.NewHarness(t)
	initial := types.AppConfig{
		MetroMakerDataPath: "first/metro",
		ExecutablePath:     "first.exe",
	}
	updated := types.AppConfig{
		MetroMakerDataPath: "second/metro",
		ExecutablePath:     "second.exe",
	}

	require.NoError(t, WriteAppConfig(initial))
	cfg := NewConfig()

	resolved, err := cfg.ResolveConfig()
	require.NoError(t, err)
	require.Equal(t, initial, resolved.Config)

	require.NoError(t, WriteAppConfig(updated))
	runtimeBeforeReload := cfg.GetConfig()
	require.Equal(t, initial, runtimeBeforeReload.Config)

	reloaded, err := cfg.ResolveConfig()
	require.NoError(t, err)
	require.Equal(t, updated, reloaded.Config)
}

func TestHasGithubTokenFlag(t *testing.T) {
	h := setup(t, types.AppConfig{
		GithubToken: "github_pat_example",
	})

	resolved := h.runtime()
	require.True(t, resolved.HasGithubToken)
	require.Empty(t, resolved.Config.GithubToken)
}

func TestUpdateAndClearGithubToken(t *testing.T) {
	h := setup(t, types.AppConfig{})

	updated := h.cfg.UpdateGithubToken("  mrao_token  ")
	require.Equal(t, types.ResponseSuccess, updated.Status)
	require.True(t, updated.HasGithubToken)
	require.Empty(t, updated.Config.GithubToken)
	require.Equal(t, "  mrao_token  ", h.cfg.GetGithubToken())

	// Runtime-only update should not mutate persisted config until SaveConfig.
	require.Equal(t, types.AppConfig{}, h.persisted())

	saved := h.cfg.SaveConfig()
	require.Equal(t, types.ResponseSuccess, saved.Status)
	// After persisting, the config should reflect the updated GitHub token
	require.Equal(t, "  mrao_token  ", h.persisted().GithubToken)

	cleared := h.cfg.ClearGithubToken()
	require.Equal(t, types.ResponseSuccess, cleared.Status)
	require.False(t, cleared.HasGithubToken)
	require.Empty(t, cleared.Config.GithubToken)
	require.Empty(t, h.cfg.GetGithubToken())
}

func mockGithubTokenValidationResponse(t *testing.T, apiKey string) func() (bool, error) {
	return func() (bool, error) {
		httpserver := http.NewServeMux()
		httpserver.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			expectedAuth := "token " + apiKey
			if authHeader != expectedAuth {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"limit": "none"}`))
			require.NoError(t, err)
		})

		openPort, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		require.NoError(t, openPort.Close())
		port := openPort.Addr().(*net.TCPAddr).Port

		testServer := http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: httpserver,
		}
		go testServer.ListenAndServe()

		request, err := http.NewRequest("GET", "http://localhost:"+strconv.Itoa(port)+"/rate_limit", nil)
		request.Header.Add("Authorization", "token "+apiKey)

		req, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		defer req.Body.Close()
		defer testServer.Close()
		return req.StatusCode == http.StatusOK, nil
	}
}

func TestGithubTokenIsValid(t *testing.T) {
	h := setup(t, types.AppConfig{})

	// Invalid token (empty)
	require.False(t, h.cfg.IsGithubTokenValid().Valid)

	// Invalid token (whitespace)
	updatedWhitespace := h.cfg.UpdateGithubToken("   ")
	require.Equal(t, types.ResponseSuccess, updatedWhitespace.Status)
	require.False(t, h.cfg.IsGithubTokenValid().Valid)

	// Token set, but invalid
	updatedInvalid := h.cfg.UpdateGithubToken("invalid_token")
	require.Equal(t, types.ResponseSuccess, updatedInvalid.Status)
	require.False(t, h.cfg.IsGithubTokenValid().Valid)

	// Valid token
	res, err := mockGithubTokenValidationResponse(t, "github_pat_example")()
	require.NoError(t, err)
	require.True(t, res)
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

	updated := h.cfg.ClearConfig()
	require.Equal(t, types.ResponseSuccess, updated.Status)
	require.Equal(t, types.AppConfig{}, updated.Config)

	runtimeConfig := h.runtime()
	require.Equal(t, types.AppConfig{}, runtimeConfig.Config)

	// ClearConfig should only affect the runtime config; no mutation should occur to the persisted config
	require.Equal(t, original, h.persisted())
}

func TestFindDefaultPathReturnsFirstMatchingDirectory(t *testing.T) {
	candidatePaths := testCandidatePaths(t)
	found, success := FindDefaultPath(candidatePaths, true)
	require.True(t, success)
	require.Equal(t, candidatePaths[3], found)
}

func TestFindDefaultPathReturnsFirstMatchingFile(t *testing.T) {
	candidatePaths := testCandidatePaths(t)
	found, success := FindDefaultPath(candidatePaths, false)
	require.True(t, success)
	require.Equal(t, candidatePaths[2], found)
}

func TestFindDefaultPathReturnsNotFoundWhenTypeMismatches(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "candidate.exe")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	// Executable file does not match when looking for directory
	found, success := FindDefaultPath([]string{filePath}, true)
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

	_, err := h.cfg.UpdateMetroMakerDataFolder(metroMakerPath)
	require.NoError(t, err)
	saved := h.cfg.SaveConfig()
	require.Equal(t, types.ResponseSuccess, saved.Status)
	detectedPath := createWritableCandidateFile(t, DefaultExecutableCandidates())

	response := h.cfg.OpenExecutableDialog(types.SetConfigPathOptions{AllowAutoDetect: true})
	require.Equal(t, types.ResponseSuccess, response.Status)
	require.Equal(t, types.SourceAutoDetected, response.Result.SetConfigSource)
	require.Equal(t, detectedPath, response.Result.AutoDetectedPath)
	require.Equal(t, detectedPath, response.Result.ResolveConfigResult.Config.ExecutablePath)

	runtimeCfg := h.runtime()
	require.Equal(t, detectedPath, runtimeCfg.Config.ExecutablePath)

	require.Equal(t, types.AppConfig{
		MetroMakerDataPath: metroMakerPath,
	}, h.persisted())
}

func TestOpenMetroMakerDialogAutoDetectSuccessDoesNotPersist(t *testing.T) {
	h := setup(t, types.AppConfig{})
	executablePath := createWritableCandidateFile(t, DefaultExecutableCandidates())

	_, err := h.cfg.UpdateExecutable(executablePath)
	require.NoError(t, err)
	saved := h.cfg.SaveConfig()
	require.Equal(t, types.ResponseSuccess, saved.Status)
	detectedPath := createWritableCandidateDir(t, DefaultMetroMakerDataFolderCandidates())

	response := h.cfg.OpenMetroMakerDataFolderDialog(types.SetConfigPathOptions{AllowAutoDetect: true})
	require.Equal(t, types.ResponseSuccess, response.Status)
	require.Equal(t, types.SourceAutoDetected, response.Result.SetConfigSource)
	require.Equal(t, detectedPath, response.Result.AutoDetectedPath)
	require.Equal(t, detectedPath, response.Result.ResolveConfigResult.Config.MetroMakerDataPath)

	runtimeCfg := h.runtime()
	require.Equal(t, detectedPath, runtimeCfg.Config.MetroMakerDataPath)

	require.Equal(t, types.AppConfig{
		ExecutablePath: executablePath,
	}, h.persisted())
}

func TestTryAutoDetectExecutableSucceedsWhenExecutablePathIsValid(t *testing.T) {
	testutil.NewHarness(t)
	detectedPath := createWritableCandidateFile(t, DefaultExecutableCandidates())

	cfg := NewConfig()
	autoDetected, success := cfg.TryAutoDetectPath(
		DefaultExecutableCandidates(),
		false,
		cfg.UpdateExecutable,
		func(v types.ConfigPathValidation) bool { return v.ExecutablePathValid },
	)
	require.True(t, success)
	require.Equal(t, types.SourceAutoDetected, autoDetected.SetConfigSource)
	require.Equal(t, detectedPath, autoDetected.AutoDetectedPath)
	require.Equal(t, detectedPath, autoDetected.ResolveConfigResult.Config.ExecutablePath)

	runtimeAfter := cfg.GetConfig()
	require.Equal(t, detectedPath, runtimeAfter.Config.ExecutablePath)
}

func TestTryAutoDetectMetroMakerSucceedsWhenMetroMakerDataPathIsValid(t *testing.T) {
	testutil.NewHarness(t)
	detectedPath := createWritableCandidateDir(t, DefaultMetroMakerDataFolderCandidates())

	cfg := NewConfig()
	autoDetected, success := cfg.TryAutoDetectPath(
		DefaultMetroMakerDataFolderCandidates(),
		true,
		cfg.UpdateMetroMakerDataFolder,
		func(v types.ConfigPathValidation) bool { return v.MetroMakerDataPathValid },
	)
	require.True(t, success)
	require.Equal(t, types.SourceAutoDetected, autoDetected.SetConfigSource)
	require.Equal(t, detectedPath, autoDetected.AutoDetectedPath)
	require.Equal(t, detectedPath, autoDetected.ResolveConfigResult.Config.MetroMakerDataPath)

	runtimeAfter := cfg.GetConfig()
	require.Equal(t, detectedPath, runtimeAfter.Config.MetroMakerDataPath)
}
