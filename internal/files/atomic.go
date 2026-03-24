package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type AtomicFileWrite struct {
	Path  string
	Label string
	Data  []byte
	Perm  os.FileMode
}

type atomicWriteArgs struct {
	spec         AtomicFileWrite
	tempPath     string
	backupPath   string
	hasOriginal  bool
	isCommitted  bool
	backupExists bool
}

// WriteFilesAtomically writes a batch of files to disk with best-effort all-or-nothing semantics.
// It writes each file to a temp file first, then commits with backup/rollback to avoid partial update on errors.
func WriteFilesAtomically(writes []AtomicFileWrite) error {
	if len(writes) == 0 {
		return nil
	}

	prepared := make([]atomicWriteArgs, 0, len(writes))
	for _, write := range writes {
		nextWrite, err := prepareAtomicWrite(write)
		if err != nil {
			cleanupPrepared(prepared)
			return err
		}
		prepared = append(prepared, nextWrite)
	}

	for i := range prepared {
		if err := commitPreparedWrite(&prepared[i]); err != nil {
			rollbackPrepared(prepared[:i+1])
			cleanupBackups(prepared)
			cleanupPrepared(prepared)
			return err
		}
	}

	cleanupBackups(prepared)
	cleanupPrepared(prepared)
	return nil
}

// prepareAtomicWrite prepares an atomic write by creating a temp file with the intended content and permissions for the target path, and checking for existing files to determine if backup is needed.
func prepareAtomicWrite(write AtomicFileWrite) (atomicWriteArgs, error) {
	if write.Path == "" {
		return atomicWriteArgs{}, fmt.Errorf("atomic write path cannot be empty for %q", write.Label)
	}
	if write.Perm == 0 {
		write.Perm = 0o644
	}
	if err := recoverAtomicBackup(write.Path, write.Label); err != nil {
		return atomicWriteArgs{}, err
	}

	// Ensure the target directory exists before creating temp files.
	dir := filepath.Dir(write.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create directory for %s %q: %w", write.Label, write.Path, err)
	}

	tempFile, err := os.CreateTemp(dir, "."+filepath.Base(write.Path)+".tmp-*")
	if err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create temp file for %s %q: %w", write.Label, write.Path, err)
	}

	// For subsequent operations, ensure the temp file is cleaned up to avoid littering the filesystem with orphaned temp files.
	failWithCleanup := func(format string, innerErr error) (atomicWriteArgs, error) {
		closeAndRemoveTempFile(tempFile)
		return atomicWriteArgs{}, fmt.Errorf(format, write.Label, write.Path, innerErr)
	}

	if err := tempFile.Chmod(write.Perm); err != nil {
		return failWithCleanup("failed to set temp file mode for %s %q: %w", err)
	}

	if _, err := tempFile.Write(write.Data); err != nil {
		return failWithCleanup("failed to write temp data for %s %q: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return failWithCleanup("failed to fsync temp data for %s %q: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempFile.Name())
		return atomicWriteArgs{}, fmt.Errorf("failed to close temp file for %s %q: %w", write.Label, write.Path, err)
	}

	return atomicWriteArgs{
		spec:     write,
		tempPath: tempFile.Name(),
	}, nil
}

// closeAndRemoveTempFile attempts to close and remove a temp file, ignoring errors since if entered, the system is already in an error state.
func closeAndRemoveTempFile(tempFile *os.File) {
	if tempFile == nil {
		return
	}
	_ = tempFile.Close()
	_ = os.Remove(tempFile.Name())
}

// recoverAtomicBackup checks for the existence of the target file and its backup, and attempts to restore from backup if the target is missing.
func recoverAtomicBackup(path string, label string) error {
	backupPath := path + ".bak"
	_, targetPathErr := os.Stat(path)
	_, backupPathErr := os.Stat(backupPath)

	// If the target file is missing but a backup exists, attempt to recover by restoring the backup.
	if errors.Is(targetPathErr, fs.ErrNotExist) && backupPathErr == nil {
		if err := os.Rename(backupPath, path); err != nil {
			return fmt.Errorf("failed to recover backup for %s %q: %w", label, path, err)
		}
		return nil
	}

	// If both files exist or both are missing, attempt to clean up any existing backup to avoid confusion on next operations.
	if targetPathErr == nil && backupPathErr == nil {
		_ = os.Remove(backupPath)
	}

	if targetPathErr != nil && !errors.Is(targetPathErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect %s %q for backup recovery: %w", label, path, targetPathErr)
	}
	if backupPathErr != nil && !errors.Is(backupPathErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect backup for %s %q: %w", label, path, backupPathErr)
	}
	return nil
}

// commitPreparedWrite replaces the target file with the prepared temp file, keeping a backup of the original if it exists.
func commitPreparedWrite(write *atomicWriteArgs) error {
	if info, err := os.Stat(write.spec.Path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s target %q is a directory", write.spec.Label, write.spec.Path)
		}
		write.hasOriginal = true
		write.backupPath = write.spec.Path + ".bak"
		_ = os.Remove(write.backupPath)
		if err := os.Rename(write.spec.Path, write.backupPath); err != nil {
			return fmt.Errorf("failed to backup %s %q: %w", write.spec.Label, write.spec.Path, err)
		}
		write.backupExists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect %s %q before commit: %w", write.spec.Label, write.spec.Path, err)
	}

	if err := os.Rename(write.tempPath, write.spec.Path); err != nil {
		if write.backupExists {
			_ = os.Rename(write.backupPath, write.spec.Path)
			write.backupExists = false
		}
		return fmt.Errorf("failed to replace %s %q atomically: %w", write.spec.Label, write.spec.Path, err)
	}

	write.isCommitted = true
	return nil
}

// rollbackPrepared attempts to restore original files from backups for any committed writes.
// This function ignores errors since if entered, the system is already in an error state.
func rollbackPrepared(prepared []atomicWriteArgs) {
	for i := len(prepared) - 1; i >= 0; i-- {
		write := prepared[i]
		if !write.isCommitted {
			continue
		}

		// If the original file existed and was backed up, restore it; otherwise, remove the target file to avoid leaving a half-updated state.
		if write.hasOriginal && write.backupPath != "" {
			_ = os.Remove(write.spec.Path)
			_ = os.Rename(write.backupPath, write.spec.Path)
			continue
		}

		_ = os.Remove(write.spec.Path)
	}
}

// cleanupPrepared removes any temp files created during the preparation phase of an atomic write batch.
func cleanupPrepared(prepared []atomicWriteArgs) {
	for _, write := range prepared {
		if write.tempPath != "" {
			_ = os.Remove(write.tempPath)
		}
	}
}

// cleanupBackups removes any backup files created during the commit phase of an atomic write batch.
func cleanupBackups(prepared []atomicWriteArgs) {
	for _, write := range prepared {
		if !write.backupExists || write.backupPath == "" {
			continue
		}
		_ = os.Remove(write.backupPath)
	}
}
