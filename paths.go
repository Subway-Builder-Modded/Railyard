package main

import (
	"os"
	"path/filepath"
)

const (
	// AppDirName is the root folder under the OS user config directory.
	AppDirName = "railyard"
	// RegistryDirName is the local git clone folder for the registry repository.
	RegistryDirName = "registry"
	// ConfigFileName is the persisted app config file name.
	ConfigFileName = "config.json"
	// InstalledModsFileName is the filename for storing installed mods info.
	InstalledModsFileName = "installed_mods.json"
	// InstalledMapsFileName is the filename for storing installed maps info.
	InstalledMapsFileName = "installed_maps.json"
	// UserProfilesFileName is the persisted user profiles file name.
	UserProfilesFileName = "user_profiles.json"
)

// UserConfigRoot resolves the base user config directory with a home-directory fallback.
func UserConfigRoot() string {
	configDir, err := os.UserConfigDir()
	if err == nil {
		return configDir
	}

	// Fallback to home directory if UserConfigDir fails
	home, _ := os.UserHomeDir()
	return home
}

// AppDataRoot returns the shared railyard folder path used by backend storage.
func AppDataRoot() string {
	return filepath.Join(UserConfigRoot(), AppDirName)
}

// RegistryRepoPath returns the local filesystem path for the cloned registry.
func RegistryRepoPath() string {
	return filepath.Join(AppDataRoot(), RegistryDirName)
}

// ConfigPath returns the default filesystem path for persisted app config.
func ConfigPath() string {
	return filepath.Join(AppDataRoot(), ConfigFileName)
}

// TilesPath returns the default filesystem path for cached map tiles.
func TilesPath() string {
	return filepath.Join(AppDataRoot(), "tiles")
}

func InstalledModsPath() string {
	return filepath.Join(AppDataRoot(), InstalledModsFileName)
}

func InstalledMapsPath() string {
	return filepath.Join(AppDataRoot(), InstalledMapsFileName)
	
// UserProfilesPath returns the default filesystem path for persisted user profiles.
func UserProfilesPath() string {
	return filepath.Join(AppDataRoot(), UserProfilesFileName)
}
