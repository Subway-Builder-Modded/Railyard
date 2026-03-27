package files

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"railyard/internal/logger"
	"railyard/internal/types"
)

// CopyDirFromFS ensures the destination directory exists and copies all files from sourceFS into it.
func CopyDirFromFS(destDir string, sourceFS fs.FS) error {
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}
	return fs.WalkDir(sourceFS, ".", getWalkDirCopyFunc(destDir, sourceFS))
}

// getWalkDirCopyFunc returns a WalkDirFunc that copies each file from the sourceFS to the destination directory, preserving relative paths.
func getWalkDirCopyFunc(destDir string, sourceFS fs.FS) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(destPath, os.ModePerm)
		}

		mode := fs.FileMode(0o644)
		if info, infoErr := entry.Info(); infoErr == nil && info.Mode().Perm() != 0 {
			mode = info.Mode().Perm()
		}
		return copyFileFromFS(sourceFS, path, destPath, mode)
	}
}

// copyFileFromFS copies a single file from the sourceFS to the destination path on disk, creating any necessary parent directories.
func copyFileFromFS(sourceFS fs.FS, sourcePath string, destPath string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return err
	}

	sourceFile, err := sourceFS.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		_ = destFile.Close()
		return err
	}
	return destFile.Close()
}

// CopyOptionalFile copies a file if present; a missing source is treated as success.
func CopyOptionalFile(src, dst, profileID, mapID, fileType string, logger logger.Logger) (types.GenericResponse, bool) {
	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return types.GenericResponse{}, true
		}
		logger.Error(fmt.Sprintf("Failed to stat optional %s file", fileType), err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to stat optional %s file: %w", fileType, err).Error()), false
	}
	return CopyFileWithDest(src, dst, profileID, mapID, fileType, logger)
}
