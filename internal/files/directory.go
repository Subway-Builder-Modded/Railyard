package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// DirectorySize returns the total size, in bytes, of all files under dirPath.
func DirectorySize(dirPath string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dirPath, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

// ManagedDirectorySize returns DirectorySize for a directory only when the directory and marker file both exist.
func ManagedDirectorySize(dirPath string, markerFileName string) (int64, error) {
	hasMarker, err := HasAssetMarker(dirPath, markerFileName)
	if err != nil {
		return 0, err
	}
	if !hasMarker {
		return 0, nil
	}

	return DirectorySize(dirPath)
}

// HasAssetMarker reports whether dirPath exists and contains markerFileName.
func HasAssetMarker(dirPath string, markerFileName string) (bool, error) {
	if _, err := os.Stat(dirPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	markerPath := filepath.Join(dirPath, markerFileName)
	if _, err := os.Stat(markerPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
