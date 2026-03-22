package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/requests"
	"railyard/internal/types"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	// Mutex should be locked for all read/write operations
	mu     sync.Mutex
	ctx    context.Context
	Cfg    types.AppConfig
	logger logger.Logger
	loaded bool
}

func NewConfig(l logger.Logger) *Config {
	if l == nil {
		panic("config.NewConfig requires non-nil logger")
	}
	return &Config{logger: l}
}

func (s *Config) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

func ReadAppConfig() (types.AppConfig, error) {
	return files.ReadJSON[types.AppConfig](
		paths.ConfigPath(),
		"app config",
		files.JSONReadOptions{
			AllowMissing: true,
			AllowEmpty:   true,
		},
	)
}

func WriteAppConfig(cfg types.AppConfig) error {
	return files.WriteJSON(paths.ConfigPath(), "app config", cfg)
}

func resolveConfigResultFromAppConfig(cfg types.AppConfig) types.ResolveConfigResult {
	_, validation := cfg.ValidateConfigPaths()
	redacted := cfg
	redacted.GithubToken = ""
	return types.ResolveConfigResult{
		Config:         redacted,
		Validation:     validation,
		HasGithubToken: cfg.GithubToken != "",
	}
}

// ResolveConfig returns the current config from disk, or empty defaults when missing.
// This should only be called once on app startup to initialize the in-memory config state
func (s *Config) ResolveConfig() (types.ResolveConfigResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	diskCfg, err := ReadAppConfig()
	if err != nil {
		return types.ResolveConfigResult{}, err
	}

	s.Cfg = diskCfg
	s.loaded = true

	return resolveConfigResultFromAppConfig(s.Cfg), nil
}

// GetConfig returns the current in-memory config without re-reading from disk.
func (s *Config) GetConfig() types.ResolveConfigResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("Config resolved"),
		ResolveConfigResult: resolveConfigResultFromAppConfig(s.Cfg),
	}
}

func (s *Config) UpdateConfig(mutator func(*types.AppConfig), persist bool) (types.ResolveConfigResult, error) {
	s.logger.Info("Updating config", "persist", persist)
	s.mu.Lock()
	defer s.mu.Unlock()

	mutator(&s.Cfg)
	s.loaded = true

	if persist {
		s.logger.Info("Persisting updated config to disk")
		if err := WriteAppConfig(s.Cfg); err != nil {
			s.logger.Error("Failed to write updated config to disk", err)
			return types.ResolveConfigResult{}, err
		}
	}

	s.logger.Info("Config updated successfully", "persisted", persist)
	return resolveConfigResultFromAppConfig(s.Cfg), nil
}

func (s *Config) UpdateCheckForUpdatesOnLaunch(checkForUpdates bool) types.ResolveConfigResponse {
	result, err := s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.CheckForUpdatesOnLaunch = checkForUpdates
	}, false)
	if err != nil {
		return types.ResolveConfigResponse{
			GenericResponse: types.ErrorResponse(err.Error()),
		}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("Config updated"),
		ResolveConfigResult: result,
	}
}

func (s *Config) CompleteSetup() types.ResolveConfigResponse {
	result, err := s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.SetupCompleted = true
	}, true) // Persist to disk immediately
	if err != nil {
		return types.ResolveConfigResponse{
			GenericResponse: types.ErrorResponse(err.Error()),
		}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("Setup completed"),
		ResolveConfigResult: result,
	}
}

// UpdateExecutable updates and persists ExecutablePath to the runtime app config.
func (s *Config) UpdateExecutable(executablePath string) (types.ResolveConfigResult, error) {
	return s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.ExecutablePath = strings.TrimSpace(executablePath)
	}, false)
}

// UpdateMetroMakerDataFolder updates and persists metroMakerDataPath to the runtime app config.
func (s *Config) UpdateMetroMakerDataFolder(metroMakerDataPath string) (types.ResolveConfigResult, error) {
	return s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.MetroMakerDataPath = strings.TrimSpace(metroMakerDataPath)
	}, false)
}

// SetConfig replaces the runtime app config with the provided object.
func (s *Config) SetConfig(next types.AppConfig) (types.AppConfig, error) {
	updated, err := s.UpdateConfig(func(cfg *types.AppConfig) {
		*cfg = types.AppConfig{
			MetroMakerDataPath:      strings.TrimSpace(next.MetroMakerDataPath),
			ExecutablePath:          strings.TrimSpace(next.ExecutablePath),
			GithubToken:             next.GithubToken,
			CheckForUpdatesOnLaunch: next.CheckForUpdatesOnLaunch,
			SetupCompleted:          next.SetupCompleted,
		}
	}, false)
	if err != nil {
		return types.AppConfig{}, err
	}

	return updated.Config, nil
}

// ClearConfig clears all config fields in memory (by replacing them with zero values).
func (s *Config) ClearConfig() types.ResolveConfigResponse {
	updated, err := s.SetConfig(types.AppConfig{})
	if err != nil {
		return types.ResolveConfigResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("Config cleared"),
		ResolveConfigResult: resolveConfigResultFromAppConfig(updated),
	}
}

// SaveConfig persists the current runtime config state to disk.
func (s *Config) SaveConfig() types.ResolveConfigResponse {
	result, err := s.UpdateConfig(func(*types.AppConfig) {}, true)
	if err != nil {
		return types.ResolveConfigResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("Config saved"),
		ResolveConfigResult: result,
	}
}

func (s *Config) UpdateGithubToken(githubToken string) types.ResolveConfigResponse {
	result, err := s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.GithubToken = githubToken
	}, false)
	if err != nil {
		return types.ResolveConfigResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("GitHub token updated"),
		ResolveConfigResult: result,
	}
}

func (s *Config) ClearGithubToken() types.ResolveConfigResponse {
	result, err := s.UpdateConfig(func(cfg *types.AppConfig) {
		cfg.GithubToken = ""
	}, false)
	if err != nil {
		return types.ResolveConfigResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.ResolveConfigResponse{
		GenericResponse:     types.SuccessResponse("GitHub token cleared"),
		ResolveConfigResult: result,
	}
}

func (s *Config) GetGithubToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Cfg.GithubToken
}

func (s *Config) IsGithubTokenValid() types.GithubTokenValidResponse {
	s.logger.Info("Validating GitHub token")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Cfg.GithubToken == "" {
		s.logger.Warn("GitHub token is not set, not validating")
		return types.GithubTokenValidResponse{
			GenericResponse: types.SuccessResponse("GitHub token not set"),
			Valid:           false,
		}
	}

	resp, err := requests.GetWithGithubToken(http.DefaultClient, requests.GithubTokenRequestArgs{
		URL:         "https://api.github.com/rate_limit",
		GitHubToken: s.Cfg.GithubToken,
		OnTokenRejected: func(statusCode int) {
			s.logger.Warn("GitHub token rejected during validation", "status", statusCode)
		},
	})

	if err != nil {
		s.logger.Error("Failed to validate GitHub token", err)
		errorResponse := types.ErrorResponse(fmt.Sprintf("failed to validate GitHub token: %v", err))
		if apiErrorType, apiErrorSource, apiStatusCode, ok := requests.ResolveAPIError(err); ok {
			errorResponse.APIErrorType = apiErrorType
			errorResponse.APIErrorSource = apiErrorSource
			errorResponse.APIStatusCode = apiStatusCode
		}
		return types.GithubTokenValidResponse{
			GenericResponse: errorResponse,
			Valid:           false,
		}
	}
	defer resp.Body.Close()

	s.logger.Info("GitHub token validation response received", "status", resp.StatusCode)
	// If GitHub rejected the token, requests.GetWithGithubToken may transparently
	// retry unauthenticated. In that case the token is still invalid.
	if resp.Request != nil && resp.Request.Header.Get("Authorization") == "" && s.Cfg.GithubToken != "" {
		return types.GithubTokenValidResponse{
			GenericResponse: types.SuccessResponse("GitHub token invalid"),
			Valid:           false,
		}
	}

	return types.GithubTokenValidResponse{
		GenericResponse: types.SuccessResponse("GitHub token validated"),
		Valid:           resp.StatusCode == http.StatusOK,
	}
}

/* ===== Dialog Functions ===== */

// OpenMetroMakerDataFolderDialog opens a directory picker and persists MetroMakerDataPath when selected.
// On user cancel, it returns the current config unchanged.
func (s *Config) OpenMetroMakerDataFolderDialog(options types.SetConfigPathOptions) types.SetConfigPathResponse {
	result, err := s.setConfigPathWithDialog(
		options,
		func() (types.SetConfigPathResult, bool) {
			return s.TryAutoDetectPath(
				DefaultMetroMakerDataFolderCandidates(),
				true,
				s.UpdateMetroMakerDataFolder,
				func(v types.ConfigPathValidation) bool { return v.MetroMakerDataPathValid },
			)
		},
		func(ctx context.Context) (string, error) {
			return wruntime.OpenDirectoryDialog(ctx, wruntime.OpenDialogOptions{
				Title:                "Select MetroMaker Data Folder",
				DefaultDirectory:     paths.AppDataRoot(),
				CanCreateDirectories: false,
			})
		},
		s.UpdateMetroMakerDataFolder,
	)
	if err != nil {
		return types.SetConfigPathResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.SetConfigPathResponse{
		GenericResponse: types.SuccessResponse("Config path resolved"),
		Result:          result,
	}
}

// OpenExecutableDialog opens a file picker and persists ExecutablePath when selected.
// On user cancel, it returns the current config unchanged.
func (s *Config) OpenExecutableDialog(options types.SetConfigPathOptions) types.SetConfigPathResponse {
	result, err := s.setConfigPathWithDialog(
		options,
		func() (types.SetConfigPathResult, bool) {
			return s.TryAutoDetectPath(
				DefaultExecutableCandidates(),
				false,
				s.UpdateExecutable,
				func(v types.ConfigPathValidation) bool { return v.ExecutablePathValid },
			)
		},
		func(ctx context.Context) (string, error) {
			return wruntime.OpenFileDialog(ctx, wruntime.OpenDialogOptions{
				Title:            "Select Executable",
				DefaultDirectory: DefaultExecutableDialogDirectory(),
				Filters: []wruntime.FileFilter{
					{
						DisplayName: "All Files",
						Pattern:     "*",
					},
				},
			})
		},
		s.UpdateExecutable,
	)
	if err != nil {
		return types.SetConfigPathResponse{GenericResponse: types.ErrorResponse(err.Error())}
	}
	return types.SetConfigPathResponse{
		GenericResponse: types.SuccessResponse("Config path resolved"),
		Result:          result,
	}
}

func (s *Config) setConfigPathWithDialog(
	options types.SetConfigPathOptions,
	autoDetect func() (types.SetConfigPathResult, bool),
	dialog func(ctx context.Context) (string, error),
	pathMutation func(path string) (types.ResolveConfigResult, error),
) (types.SetConfigPathResult, error) {
	s.logger.Info("Setting config path with dialog")
	if options.AllowAutoDetect { // If auto-detection is allowed, attempt to find a valid path before showing the dialog
		s.logger.Info("Attempting auto-detection of config path")
		autoDetected, ok := autoDetect()
		if ok {
			s.logger.Info("Auto-detection successful", "path", autoDetected.AutoDetectedPath)
			return autoDetected, nil
		}
		s.logger.Info("Auto-detection failed, showing dialog")
	}

	selectedPath, err := dialog(s.ctx)
	if err != nil {
		s.logger.Error("Dialog resulted in an error", err)
		return types.SetConfigPathResult{}, err
	}

	// User cancellation results in an empty path
	if strings.TrimSpace(selectedPath) == "" {
		s.logger.Info("Dialog cancelled by user, returning current config")
		current := s.GetConfig()
		return types.SetConfigPathResult{
			ResolveConfigResult: current.ResolveConfigResult,
			SetConfigSource:     types.SourceCancelled,
		}, nil
	}

	updated, updateErr := pathMutation(selectedPath)
	if updateErr != nil {
		s.logger.Error("Failed to update config with selected path", updateErr, "selectedPath", selectedPath)
		return types.SetConfigPathResult{}, updateErr
	}

	s.logger.Info("Config path updated successfully", "newPath", selectedPath)
	return types.SetConfigPathResult{
		ResolveConfigResult: updated,
		SetConfigSource:     types.SourceDialogSelected,
	}, nil
}

/* ===== Auto-detection logic and helpers ===== */

func (s *Config) TryAutoDetectPath(
	candidates []string,
	shouldBeDir bool,
	updatePath func(path string) (types.ResolveConfigResult, error),
	isPathValid func(types.ConfigPathValidation) bool,
) (types.SetConfigPathResult, bool) {
	detectedPath, success := FindDefaultPath(candidates, shouldBeDir)
	if !success {
		s.logger.Info("Auto-detection failed")
		return types.SetConfigPathResult{}, false
	}

	resolved, err := updatePath(detectedPath)
	if err != nil {
		s.logger.Error("Failed to resolve auto-detected path", err, "path", detectedPath)
		return types.SetConfigPathResult{}, false
	}
	if !isPathValid(resolved.Validation) {
		s.logger.Warn("Auto-detected path is not valid", "path", detectedPath)
		return types.SetConfigPathResult{}, false
	}

	s.logger.Info("Auto-detected path is valid", "path", detectedPath)
	return types.SetConfigPathResult{
		ResolveConfigResult: resolved,
		SetConfigSource:     types.SourceAutoDetected,
		AutoDetectedPath:    detectedPath,
	}, true
}

// FindDefaultPath iterates through the provided candidates and returns the first path that exists and matches the expected type (file vs directory).
func FindDefaultPath(candidates []string, shouldBeDir bool) (detectedPath string, success bool) {
	for _, candidate := range candidates {
		if candidate == "" || !filepath.IsAbs(candidate) {
			continue
		}

		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		if shouldBeDir == info.IsDir() {
			return candidate, true
		}
	}

	return "", false
}

// DefaultMetroMakerDataFolderCandidates returns the default locations for the metro maker data folder
func DefaultMetroMakerDataFolderCandidates() []string {
	return []string{
		filepath.Join(paths.UserConfigRoot(), "metro-maker4"),
	}
}

// DefaultExecutableCandidates returns known default locations for the executable, based on OS conventions and the common install patterns of applications on each platform.
func DefaultExecutableCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return []string{
			filepath.Join("/Applications", "Subway Builder.app", "Contents", "MacOS", "Subway Builder"),
			filepath.Join(homeDir, "Applications", "Subway Builder.app", "Contents", "MacOS", "Subway Builder"),
		}
	case "windows":
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		programFiles := strings.TrimSpace(os.Getenv("ProgramFiles"))
		programFilesX86 := strings.TrimSpace(os.Getenv("ProgramFiles(x86)"))

		return []string{
			filepath.Join(localAppData, "Programs", "Subway Builder", "Subway Builder.exe"),
			filepath.Join(programFiles, "Subway Builder", "Subway Builder.exe"),
			filepath.Join(programFilesX86, "Subway Builder", "Subway Builder.exe"),
		}
	case "linux":
		homeDir, _ := os.UserHomeDir()
		return []string{
			filepath.Join(homeDir, "Applications", "Subway Builder.AppImage"),
			filepath.Join(homeDir, ".local", "bin", "Subway Builder.AppImage"),
			filepath.Join("/usr", "local", "bin", "Subway Builder.AppImage"),
		}
	default:
		return nil
	}
}

func DefaultExecutableDialogDirectory() string {
	switch runtime.GOOS {
	case "darwin":
		// For MacOS, the executable could also be within ~/Applications, but here we default to system-wide Applications
		return "/Applications"
	case "windows":
		// For Windows, default to ProgramFiles, with fallbacks to ProgramFiles(x86) and then the AppData root if neither are available
		if programFiles := strings.TrimSpace(os.Getenv("ProgramFiles")); programFiles != "" {
			return programFiles
		}
		if programFilesX86 := strings.TrimSpace(os.Getenv("ProgramFiles(x86)")); programFilesX86 != "" {
			return programFilesX86
		}
		return paths.UserConfigRoot()
	case "linux":
		// If Railyard is running as AppImage, default to the same directory; otherwise, default to /usr/bin
		if appImage := strings.TrimSpace(os.Getenv("APPIMAGE")); appImage != "" {
			return filepath.Dir(appImage)
		}
		return "/usr/bin"
	default:
		return paths.UserConfigRoot()
	}
}
