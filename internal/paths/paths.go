package paths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	// LogFileName is the log file name.
	LogFileName = "railyard.log"
	// PrevLogFileName is the previous log file name.
	PrevLogFileName = "railyard.old.log"
	// LockFile is the name of the temporary file created to check if the app is already running.
	LockFile = "railyard.lock"
	// StagingDirName is the directory under a managed root used for atomic-write staging artifacts.
	StagingDirName = "railyard_staging"
	// MetroMakerRailyardDirName is the hidden directory under MetroMaker data root for Railyard-managed metadata.
	MetroMakerRailyardDirName = ".railyard"
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

func ProfileArchivesPath() string {
	return filepath.Join(AppDataRoot(), "profile_archives")
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
}

// UserProfilesPath returns the default filesystem path for persisted user profiles.
func UserProfilesPath() string {
	return filepath.Join(AppDataRoot(), UserProfilesFileName)
}

func LogFilePath() string {
	return filepath.Join(AppDataRoot(), LogFileName)
}

func PrevLogFilePath() string {
	return filepath.Join(AppDataRoot(), PrevLogFileName)
}

func LockFilePath() string {
	return filepath.Join(AppDataRoot(), LockFile)
}

// AppTmpStagingPath returns the managed temporary staging directory under AppData.
func AppTmpStagingPath() string {
	return filepath.Join(AppDataRoot(), StagingDirName)
}

// MetroMakerTmpStagingPath returns the managed temporary staging directory under MetroMaker data root.
func MetroMakerTmpStagingPath(metroMakerDataPath string) string {
	return JoinLocalPath(metroMakerDataPath, MetroMakerRailyardDirName, StagingDirName)
}

// MetroMakerModsPath returns the mods directory under MetroMaker data root.
func MetroMakerModsPath(metroMakerDataPath string) string {
	return JoinLocalPath(metroMakerDataPath, "mods")
}

// MetroMakerMapsDataPath returns the cities/data directory under MetroMaker data root.
func MetroMakerMapsDataPath(metroMakerDataPath string) string {
	return JoinLocalPath(metroMakerDataPath, "cities", "data")
}

// NormalizeLocalPath normalizes a local filesystem path for the current OS.
// It trims whitespace and replaces both backslashes and forward slashes with os.PathSeparator
func NormalizeLocalPath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	// Normalize backslashes to forward slashes first, then convert to
	// the OS separator, then clean.
	unified := strings.ReplaceAll(trimmed, `\`, "/")
	return filepath.Clean(filepath.FromSlash(unified))
}

// JoinLocalPath joins path segments and normalizes separators for the current OS.
func JoinLocalPath(base string, parts ...string) string {
	all := append([]string{base}, parts...)
	return NormalizeLocalPath(filepath.Join(all...))
}

// GetQuarantinePath returns the "quarantined" path for a target file using the current unix timestamp.
// This can be used to move invalid/corrupted files away from their expected location while still leaving them accessible for debugging
// Example: "user_profiles.json" -> "user_profiles.invalid.<unix>.json".
func GetQuarantinePath(targetPath string) string {
	dir := filepath.Dir(targetPath)
	ext := filepath.Ext(targetPath)
	base := strings.TrimSuffix(filepath.Base(targetPath), ext)
	name := fmt.Sprintf("%s.invalid.%d%s", base, time.Now().UnixNano(), ext)
	return filepath.Join(dir, name)
}

// QuarantineFile moves a file to a quarantined path. If the source file is missing, it is treated as a no-op.
// The logger parameter is optional and can be nil.
func QuarantineFile(sourcePath string, logger interface {
	Error(msg string, err error, attrs ...any)
	Warn(msg string, attrs ...any)
}) (success bool, backupPath string) {
	path := GetQuarantinePath(sourcePath)
	if err := os.Rename(sourcePath, path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, ""
		}
		if logger != nil {
			logger.Error("Failed to quarantine file", err, "sourcePath", sourcePath, "backupPath", path)
		}
		return false, ""
	}
	if logger != nil {
		logger.Warn("Quarantined file", "sourcePath", sourcePath, "backupPath", path)
	}
	return true, path
}

// MoveLogFile rotates the current log file to the previous log file path.
func MoveLogFile() error {
	if removeErr := os.Remove(PrevLogFilePath()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove previous log file: %w", removeErr)
	}
	if err := os.Rename(LogFilePath(), PrevLogFilePath()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to rotate log file: %w", err)
	}
	return nil
}
