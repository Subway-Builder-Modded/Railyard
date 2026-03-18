package types

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"railyard/internal/paths"
)

// AppConfig is persisted at ConfigPath() and is used for global configuration
type AppConfig struct {
	MetroMakerDataPath      string `json:"metroMakerDataPath,omitempty"`
	ExecutablePath          string `json:"executablePath,omitempty"`
	GithubToken             string `json:"githubToken,omitempty"`
	CheckForUpdatesOnLaunch bool   `json:"checkForUpdatesOnLaunch"`
	SetupCompleted          bool   `json:"setupCompleted"`
	ChromeSandboxPath       string `json:"chromeSandboxPath,omitempty"`
	// Other fields to be appended here
}

// ConfigPathValidation is the result of validating AppConfig paths
type ConfigPathValidation struct {
	IsConfigured            bool `json:"isConfigured"`
	MetroMakerDataPathValid bool `json:"metroMakerDataPathValid"`
	ExecutablePathValid     bool `json:"executablePathValid"`
}

// ResolveConfigResult describes the result of resolving app config from disk.
type ResolveConfigResult struct {
	Config         AppConfig            `json:"config"`
	Validation     ConfigPathValidation `json:"validation"`
	HasGithubToken bool                 `json:"hasGithubToken"`
}

type ResolveConfigResponse struct {
	GenericResponse
	ResolveConfigResult
}

type SetConfigSource string

const (
	SourceAutoDetected   SetConfigSource = "auto_detected"   // For when a path is automatically detected by the app
	SourceDialogSelected SetConfigSource = "dialog_selected" // For when a path is selected by the user through a dialog
	SourceCancelled      SetConfigSource = "cancelled"       // For when user cancels the dialog
)

type SetConfigPathOptions struct {
	AllowAutoDetect bool `json:"allowAutoDetect"`
}

type SetConfigPathResult struct {
	ResolveConfigResult ResolveConfigResult `json:"resolveConfigResult"`
	SetConfigSource     SetConfigSource     `json:"source"`
	AutoDetectedPath    string              `json:"autoDetectedPath,omitempty"`
}

type SetConfigPathResponse struct {
	GenericResponse
	Result SetConfigPathResult `json:"result"`
}

type GithubTokenValidResponse struct {
	GenericResponse
	Valid bool `json:"valid"`
}

// AreConfigPathsConfigured checks if both required paths have been set in AppConfig
func (c AppConfig) AreConfigPathsConfigured() bool {
	return strings.TrimSpace(c.MetroMakerDataPath) != "" && strings.TrimSpace(c.ExecutablePath) != ""
}

// GetModsFolderPath returns the full path to the mods folder based on the MetroMakerDataPath in AppConfig, or an empty string if paths are not properly configured.
func (c AppConfig) GetModsFolderPath() string {
	pathsValid, _ := c.ValidateConfigPaths()
	if pathsValid {
		return paths.JoinLocalPath(c.MetroMakerDataPath, "mods")
	}
	return ""
}

// GetThumbnailFolderPath returns the full path to the thumbnail folder based on the MetroMakerDataPath in AppConfig, or an empty string if paths are not properly configured.
func (c AppConfig) GetThumbnailFolderPath() string {
	pathsValid, _ := c.ValidateConfigPaths()
	if pathsValid {
		return paths.JoinLocalPath(c.MetroMakerDataPath, "public", "data", "city-maps")
	}
	return ""
}

// GetMapsFolderPath returns the full path to the maps folder based on the MetroMakerDataPath in AppConfig, or an empty string if paths are not properly configured.
func (c AppConfig) GetMapsFolderPath() string {
	pathsValid, _ := c.ValidateConfigPaths()
	if pathsValid {
		return paths.JoinLocalPath(c.MetroMakerDataPath, "cities", "data")
	}
	return ""
}

// isExecutable is a stricter validation than checking if a particular path is a file
// It checks if the file is a regular file and has executable permissions (or .exe extension on Windows)
func isExecutable(path string, info os.FileInfo) bool {
	if info.IsDir() || !info.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Ext(path), ".exe")
	}
	// unix: any execute bit set
	return info.Mode()&0o111 != 0
}

// ValidateConfigPaths checks whether the AppConfig has been configured and whether or not its specified paths exist on disk
func (c AppConfig) ValidateConfigPaths() (bool, ConfigPathValidation) {
	result := ConfigPathValidation{
		IsConfigured: c.AreConfigPathsConfigured(),
	}

	if strings.TrimSpace(c.MetroMakerDataPath) != "" {
		modInfo, modErr := os.Stat(c.MetroMakerDataPath)
		result.MetroMakerDataPathValid = modErr == nil && modInfo.IsDir()
	}

	if strings.TrimSpace(c.ExecutablePath) != "" {
		exeInfo, exeErr := os.Stat(c.ExecutablePath)
		result.ExecutablePathValid = exeErr == nil && isExecutable(c.ExecutablePath, exeInfo)
	}

	return result.IsValid(), result
}

func (v ConfigPathValidation) IsValid() bool {
	return v.IsConfigured && v.MetroMakerDataPathValid && v.ExecutablePathValid
}
