package files

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"railyard/internal/logger"
	"railyard/internal/types"
	"strings"
)

// WriteArchiveStream writes an archive entry stream to a destination path, conditionally using gzip compression based on the useGzip flag.
func WriteArchiveStream(destinationPath string, src io.Reader, useGzip bool) error {
	destFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if useGzip {
		gzipWriter := gzip.NewWriter(destFile)
		defer gzipWriter.Close()
		_, err = io.Copy(gzipWriter, src)
		return err
	}

	_, err = io.Copy(destFile, src)
	return err
}

func stageArchive(destinationPath string, src io.Reader, useGzip bool) (string, error) {
	tempFile, err := createManagedTempFile(destinationPath)
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if closeErr := tempFile.Close(); closeErr != nil {
		_ = os.Remove(tempPath)
		return "", closeErr
	}

	if err := WriteArchiveStream(tempPath, src, useGzip); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}

	return tempPath, nil
}

// StageArchiveForAtomicWrite opens an archive entry, stages it to a temporary file and returns the staged path for later atomic commit.
func StageArchiveForAtomicWrite(destinationPath string, entry interface {
	Open() (io.ReadCloser, error)
}, useGzip bool) (string, error) {
	src, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	return stageArchive(destinationPath, src, useGzip)
}

// CopyFileToArchive adds a file to the tar archive
func CopyFileToArchive(archive *tar.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(filepath.Dir(filePath), filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}
	header.Name = relPath

	if err := archive.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(archive, file)
	return err
}

// AddDirToArchive recursively adds a directory to the tar archive
func AddDirToArchive(archive *tar.Writer, dirPath, basePath string) error {
	return filepath.Walk(dirPath, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(basePath, filePath)
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := archive.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(archive, file)
		return err
	})
}

// ExtractArchiveToDir extracts a tar archive to a directory
func ExtractArchiveToDir(archivePath, destDir string) error {
	rawReader, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer rawReader.Close()

	archive := tar.NewReader(rawReader)

	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
			return err
		}

		targetFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, archive); err != nil {
			return err
		}
	}

	return nil
}

// ReadJSONFromTarArchive reads and decodes a JSON file from a tar archive by file basename.
// Returns found=false when the target file is not present in the archive.
func ReadJSONFromTarArchive[T any](archivePath string, targetFileName string) (T, bool, error) {
	var value T

	file, err := os.Open(archivePath)
	if err != nil {
		return value, false, err
	}
	defer file.Close()

	reader := tar.NewReader(file)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			return value, false, nil
		}
		if nextErr != nil {
			return value, false, nextErr
		}
		if header.FileInfo().IsDir() {
			continue
		}

		normalizedName := strings.ReplaceAll(header.Name, "\\", "/")
		if path.Base(normalizedName) != targetFileName {
			continue
		}

		raw, readErr := io.ReadAll(reader)
		if readErr != nil {
			return value, false, readErr
		}
		decoded, decodeErr := ParseJSON[T](raw, targetFileName)
		if decodeErr != nil {
			return value, false, decodeErr
		}
		return decoded, true, nil
	}
}

// WriteArchiveJSON writes a JSON payload into an archive staging directory.
func WriteArchiveJSON[T any](tempDir, fileName, label string, payload T) error {
	return WriteJSON(filepath.Join(tempDir, fileName), label, payload)
}

// CopyDirectory copies all files from src to dst
func CopyDirectory(src, dst string) error {
	return filepath.Walk(src, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if fileInfo.IsDir() {
			return os.MkdirAll(destPath, os.ModePerm)
		}

		srcFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// copyFileWithDest is a helper that copies a file with specific source and destination paths
func CopyFileWithDest(src, dst, profileID, mapID, fileType string, logger logger.Logger) (types.GenericResponse, bool) {
	srcFile, err := os.Open(src)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open %s from archive", fileType), err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to open %s from archive: %w", fileType, err).Error()), false
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		logger.Error(fmt.Sprintf("Failed to create directory for %s", fileType), err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to create directory for %s: %w", fileType, err).Error()), false
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create %s file", fileType), err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to create %s file: %w", fileType, err).Error()), false
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		logger.Error(fmt.Sprintf("Failed to copy %s", fileType), err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to copy %s: %w", fileType, err).Error()), false
	}

	return types.GenericResponse{}, true
}

// copyFile is a helper to copy a file from src to dst with error handling
func CopyFile(src, dst, profileID, mapID string, logger logger.Logger) (types.GenericResponse, bool) {
	srcFile, err := os.Open(src)
	if err != nil {
		logger.Error("Failed to open source file", err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to open source file: %w", err).Error()), false
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		logger.Error("Failed to create destination file", err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to create destination file: %w", err).Error()), false
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		logger.Error("Failed to copy file", err, "profile_id", profileID, "map_id", mapID)
		return types.ErrorResponse(fmt.Errorf("failed to copy file: %w", err).Error()), false
	}
	return types.GenericResponse{}, true
}
