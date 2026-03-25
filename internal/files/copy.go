package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"railyard/internal/logger"
	"railyard/internal/types"
)

// CopyDirFromFS ensures the destination directory exists and copies all files from sourceFS into it.
func CopyDirFromFS(destDir string, sourceFS fs.FS) error {
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}
	return os.CopyFS(destDir, sourceFS)
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
