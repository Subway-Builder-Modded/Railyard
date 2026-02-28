package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"railyard/internal/files"
	"railyard/internal/types"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	// Mutex should be locked for all read/write operations
	mu     sync.Mutex
	ctx    context.Context
	cfg    types.AppConfig
	loaded bool
}

func NewConfig() *Config {
	return &Config{}
}

func (s *Config) setContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

func readAppConfig() (types.AppConfig, error) {
	return files.ReadJSON[types.AppConfig](
		ConfigPath(),
		"app config",
		files.JSONReadOptions{
			AllowMissing: true,
			AllowEmpty:   true,
		},
	)
}

func writeAppConfig(cfg types.AppConfig) error {
	return files.WriteJSON(ConfigPath(), "app config", cfg)
}

func resolveConfigResultFromAppConfig(cfg types.AppConfig) types.ResolveConfigResult {
	_, validation := cfg.ValidateConfigPaths()
	return types.ResolveConfigResult{
		Config:     cfg,
		Validation: validation,
	}
}

// resolveConfig returns the current config from disk, or empty defaults when missing.
// This should only be called once on app startup to initialize the in-memory config state
func (s *Config) resolveConfig() (types.ResolveConfigResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	diskCfg, err := readAppConfig()
	if err != nil {
		return types.ResolveConfigResult{}, err
	}

	s.cfg = diskCfg
	s.loaded = true

	return resolveConfigResultFromAppConfig(s.cfg), nil
}

// GetConfig returns the current in-memory config without re-reading from disk.
func (s *Config) GetConfig() types.ResolveConfigResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	return resolveConfigResultFromAppConfig(s.cfg)
}

func (s *Config) updateConfig(mutator func(*types.AppConfig), persist bool) (types.ResolveConfigResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mutator(&s.cfg)
	s.loaded = true

	if persist {
		if err := writeAppConfig(s.cfg); err != nil {
			return types.ResolveConfigResult{}, err
		}
	}

	return resolveConfigResultFromAppConfig(s.cfg), nil
}

// updateExecutable updates and persists ExecutablePath to the runtime app config.
func (s *Config) updateExecutable(executablePath string) (types.ResolveConfigResult, error) {
	return s.updateConfig(func(cfg *types.AppConfig) {
		cfg.ExecutablePath = strings.TrimSpace(executablePath)
	}, false)
}

// updateMetroMakerDataFolder updates and persists metroMakerDataPath to the runtime app config.
func (s *Config) updateMetroMakerDataFolder(metroMakerDataPath string) (types.ResolveConfigResult, error) {
	return s.updateConfig(func(cfg *types.AppConfig) {
		cfg.MetroMakerDataPath = strings.TrimSpace(metroMakerDataPath)
	}, false)
}

// SetConfig replaces the runtime app config with the provided object.
func (s *Config) SetConfig(next types.AppConfig) (types.AppConfig, error) {
	updated, err := s.updateConfig(func(cfg *types.AppConfig) {
		*cfg = types.AppConfig{
			MetroMakerDataPath: strings.TrimSpace(next.MetroMakerDataPath),
			ExecutablePath:     strings.TrimSpace(next.ExecutablePath),
		}
	}, false)
	if err != nil {
		return types.AppConfig{}, err
	}

	return updated.Config, nil
}

// ClearConfig clears all config fields in memory (by replacing them with zero values).
func (s *Config) ClearConfig() (types.AppConfig, error) {
	return s.SetConfig(types.AppConfig{})
}

// SaveConfig persists the current runtime config state to disk.
func (s *Config) SaveConfig() (types.ResolveConfigResult, error) {
	return s.updateConfig(func(*types.AppConfig) {}, true)
}

/* ===== Dialog Functions ===== */

// OpenMetroMakerDataFolderDialog opens a directory picker and persists MetroMakerDataPath when selected.
// On user cancel, it returns the current config unchanged.
func (s *Config) OpenMetroMakerDataFolderDialog(options types.SetConfigPathOptions) (types.SetConfigPathResult, error) {
	return s.setConfigPathWithDialog(
		options,
		func() (types.SetConfigPathResult, bool) {
			return s.tryAutoDetectPath(
				defaultMetroMakerDataFolderCandidates(),
				true,
				s.updateMetroMakerDataFolder,
			)
		},
		func(ctx context.Context) (string, error) {
			return wruntime.OpenDirectoryDialog(ctx, wruntime.OpenDialogOptions{
				Title:                "Select MetroMaker Data Folder",
				DefaultDirectory:     AppDataRoot(),
				CanCreateDirectories: false,
			})
		},
		s.updateMetroMakerDataFolder,
	)
}

// OpenExecutableDialog opens a file picker and persists ExecutablePath when selected.
// On user cancel, it returns the current config unchanged.
func (s *Config) OpenExecutableDialog(options types.SetConfigPathOptions) (types.SetConfigPathResult, error) {
	return s.setConfigPathWithDialog(
		options,
		func() (types.SetConfigPathResult, bool) {
			return s.tryAutoDetectPath(
				defaultExecutableCandidates(),
				false,
				s.updateExecutable,
			)
		},
		func(ctx context.Context) (string, error) {
			return wruntime.OpenFileDialog(ctx, wruntime.OpenDialogOptions{
				Title:            "Select Executable",
				DefaultDirectory: defaultExecutableDialogDirectory(),
				Filters: []wruntime.FileFilter{
					{
						DisplayName: "All Files",
						Pattern:     "*",
					},
				},
			})
		},
		s.updateExecutable,
	)
}

func (s *Config) setConfigPathWithDialog(
	options types.SetConfigPathOptions,
	autoDetect func() (types.SetConfigPathResult, bool),
	dialog func(ctx context.Context) (string, error),
	pathMutation func(path string) (types.ResolveConfigResult, error),
) (types.SetConfigPathResult, error) {
	if options.AllowAutoDetect { // If auto-detection is allowed, attempt to find a valid path before showing the dialog
		autoDetected, ok := autoDetect()
		if ok {
			return autoDetected, nil
		}
	}

	selectedPath, err := dialog(s.ctx)
	if err != nil {
		return types.SetConfigPathResult{}, err
	}

	// User cancellation results in an empty path
	if strings.TrimSpace(selectedPath) == "" {
		return types.SetConfigPathResult{
			ResolveConfigResult: s.GetConfig(),
			SetConfigSource:     types.SourceCancelled,
		}, nil
	}

	updated, updateErr := pathMutation(selectedPath)
	if updateErr != nil {
		return types.SetConfigPathResult{}, updateErr
	}

	return types.SetConfigPathResult{
		ResolveConfigResult: updated,
		SetConfigSource:     types.SourceDialogSelected,
	}, nil
}

/* ===== Auto-detection logic and helpers ===== */

func (s *Config) tryAutoDetectPath(
	candidates []string,
	shouldBeDir bool,
	updatePath func(path string) (types.ResolveConfigResult, error),
) (types.SetConfigPathResult, bool) {
	detectedPath, success := findDefaultPath(candidates, shouldBeDir)
	if !success {
		return types.SetConfigPathResult{}, false
	}

	resolved, err := updatePath(detectedPath)
	if err != nil {
		return types.SetConfigPathResult{}, false
	}
	if !resolved.Validation.IsValid() {
		return types.SetConfigPathResult{}, false
	}

	return types.SetConfigPathResult{
		ResolveConfigResult: resolved,
		SetConfigSource:     types.SourceAutoDetected,
		AutoDetectedPath:    detectedPath,
	}, true
}

// findDefaultPath iterates through the provided candidates and returns the first path that exists and matches the expected type (file vs directory).
func findDefaultPath(candidates []string, shouldBeDir bool) (detectedPath string, success bool) {
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

// defaultMetroMakerDataFolderCandidates returns known default locations for the metro maker data folder
func defaultMetroMakerDataFolderCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return []string{
			filepath.Join(homeDir, "Library", "Application Support", "metro-maker4"),
		}
	case "windows":
		appData := strings.TrimSpace(os.Getenv("APPDATA"))
		return []string{
			filepath.Join(appData, "metro-maker4"),
		}
	case "linux":
		homeDir, _ := os.UserHomeDir()
		return []string{
			filepath.Join(homeDir, ".config", "metro-maker4"),
		}
	default:
		return nil
	}
}

// defaultExecutableCandidates returns known default locations for the executable, based on OS conventions and the common install patterns of applications on each platform.
func defaultExecutableCandidates() []string {
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
			filepath.Join(localAppData, "Programs", "subway-builder", "Subway Builder.exe"),
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

func defaultExecutableDialogDirectory() string {
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
		return AppDataRoot()
	case "linux":
		// If Railyard is running as AppImage, default to the same directory; otherwise, default to /usr/bin
		if appImage := strings.TrimSpace(os.Getenv("APPIMAGE")); appImage != "" {
			return filepath.Dir(appImage)
		}
		return "/usr/bin"
	default:
		return AppDataRoot()
	}
}
