package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type AtomicFileWrite struct {
	Path       string
	Label      string
	Data       []byte
	Perm       os.FileMode
	StagedPath string
}

type AtomicWrite interface {
	prepareAtomicWrite() (atomicWriteArgs, error)
}

type AtomicDirectoryWrite struct {
	Path     string
	Label    string
	Callback func(stagingPath string) error
}

type atomicWriteArgs struct {
	path         string
	label        string
	isDir        bool
	tempPath     string
	backupPath   string
	hasOriginal  bool
	isCommitted  bool
	backupExists bool
}

// StagingRoot maps a target-root prefix to a managed temporary staging root.
type StagingRoot struct {
	TargetRoot  string
	StagingRoot string
}

var (
	stagingRootsMu  sync.RWMutex
	tmpStagingRoots []StagingRoot
)

func writeTypeLabel(isDir bool) string {
	if isDir {
		return "directory"
	}
	return "file"
}

// ConfigureTmpStagingRoots sets managed staging roots for temporary writes.
func ConfigureTmpStagingRoots(roots []StagingRoot) {
	cleaned := make([]StagingRoot, len(roots))
	for i, root := range roots {
		if root.TargetRoot == "" || root.StagingRoot == "" {
			panic("files.ConfigureTmpStagingRoots requires non-empty TargetRoot and StagingRoot")
		}
		cleaned[i] = StagingRoot{
			TargetRoot:  filepath.Clean(root.TargetRoot),
			StagingRoot: filepath.Clean(root.StagingRoot),
		}
	}

	sort.SliceStable(cleaned, func(i, j int) bool {
		return len(cleaned[i].TargetRoot) > len(cleaned[j].TargetRoot)
	})

	stagingRootsMu.Lock()
	defer stagingRootsMu.Unlock()
	tmpStagingRoots = cleaned
}

// CleanupTmpStagingRoots removes all configured managed staging roots, ignoring missing paths.
func CleanupTmpStagingRoots() error {
	roots := configuredAtomicStagingRoots()
	seen := make(map[string]struct{}, len(roots))
	var errs []error

	for _, root := range roots {
		if _, ok := seen[root.StagingRoot]; ok {
			continue
		}
		seen[root.StagingRoot] = struct{}{}

		if err := os.RemoveAll(root.StagingRoot); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("failed to remove managed atomic staging root %q: %w", root.StagingRoot, err))
		}
	}

	return errors.Join(errs...)
}

func configuredAtomicStagingRoots() []StagingRoot {
	stagingRootsMu.RLock()
	defer stagingRootsMu.RUnlock()
	roots := make([]StagingRoot, len(tmpStagingRoots))
	copy(roots, tmpStagingRoots)
	return roots
}

func resolveManagedAtomicStagingDir(targetPath string) string {
	cleanTarget := filepath.Clean(targetPath)
	for _, root := range configuredAtomicStagingRoots() {
		if pathWithinRoot(cleanTarget, root.TargetRoot) {
			return root.StagingRoot
		}
	}
	return filepath.Dir(cleanTarget)
}

func pathWithinRoot(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && filepath.IsLocal(rel)
}

func createManagedTempFile(targetPath string) (*os.File, error) {
	stagingDir := resolveManagedAtomicStagingDir(targetPath)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	return os.CreateTemp(stagingDir, "."+filepath.Base(targetPath)+".tmp-*")
}

func createManagedTempDir(targetPath string) (string, error) {
	stagingDir := resolveManagedAtomicStagingDir(targetPath)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(stagingDir, "."+filepath.Base(targetPath)+".tmp-*")
}

// WriteFilesAtomically writes a batch of files to disk with best-effort all-or-nothing semantics.
// It writes each file to a temp file first, then commits with backup/rollback to avoid partial update on errors.
func WriteFilesAtomically(writes []AtomicFileWrite) error {
	pathWrites := make([]AtomicWrite, 0, len(writes))
	for _, write := range writes {
		pathWrites = append(pathWrites, write)
	}
	return WritePathsAtomically(pathWrites)
}

// WritePathsAtomically writes a batch of mixed file and directory operations with best-effort all-or-nothing semantics.
func WritePathsAtomically(writes []AtomicWrite) error {
	if len(writes) == 0 {
		return nil
	}

	prepared := make([]atomicWriteArgs, 0, len(writes))
	for _, write := range writes {
		nextWrite, err := write.prepareAtomicWrite()
		if err != nil {
			cleanupPrepared(prepared)
			return err
		}
		prepared = append(prepared, nextWrite)
	}

	return executeAtomicWrites(prepared)
}

func executeAtomicWrites(prepared []atomicWriteArgs) error {
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

// prepareAtomicWrite prepares a staging directory for an atomic directory replacement, and relies on the provided callback to populate the staging directory with the intended contents.
func (write AtomicDirectoryWrite) prepareAtomicWrite() (atomicWriteArgs, error) {
	targetPath := write.Path
	label := write.Label
	callback := write.Callback

	if targetPath == "" {
		return atomicWriteArgs{}, fmt.Errorf("atomic directory target cannot be empty for %q", label)
	}
	if callback == nil {
		return atomicWriteArgs{}, fmt.Errorf("atomic directory callback function cannot be nil for %q", label)
	}
	if err := recoverAtomicBackup(targetPath, label, true); err != nil {
		return atomicWriteArgs{}, err
	}

	parent := filepath.Dir(targetPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create parent directory for %s %q: %w", label, targetPath, err)
	}

	// Create a staging directory inside the configured managed staging root
	// for this target path to keep temp artifacts out of install directories.
	stagingPath, err := createManagedTempDir(targetPath)
	if err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create staging directory for %s %q: %w", label, targetPath, err)
	}
	// Then populate the staging directory with the provided callback, and if any error occurs during population, clean up the staging directory to avoid littering the filesystem with orphaned temp directories.
	if err := callback(stagingPath); err != nil {
		_ = os.RemoveAll(stagingPath)
		return atomicWriteArgs{}, fmt.Errorf("failed to populate staging directory for %s %q: %w", label, targetPath, err)
	}

	return atomicWriteArgs{
		path:     targetPath,
		label:    label,
		isDir:    true,
		tempPath: stagingPath,
	}, nil
}

func (write AtomicFileWrite) prepareAtomicWrite() (atomicWriteArgs, error) {
	targetPath := write.Path
	label := write.Label

	if targetPath == "" {
		return atomicWriteArgs{}, fmt.Errorf("atomic write path cannot be empty for %q", label)
	}
	if write.StagedPath != "" && len(write.Data) > 0 {
		return atomicWriteArgs{}, fmt.Errorf("atomic file write for %s %q cannot include both staged_path and inline data", label, targetPath)
	}
	if write.Perm == 0 {
		write.Perm = 0o644
	}
	if err := recoverAtomicBackup(targetPath, label, false); err != nil {
		return atomicWriteArgs{}, err
	}

	// Ensure the target directory exists before creating temp files.
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create directory for %s %q: %w", label, targetPath, err)
	}

	if write.StagedPath != "" {
		stagedInfo, err := os.Stat(write.StagedPath)
		if err != nil {
			return atomicWriteArgs{}, fmt.Errorf("failed to inspect staged file for %s %q: %w", label, targetPath, err)
		}
		if stagedInfo.IsDir() {
			return atomicWriteArgs{}, fmt.Errorf("staged file path is a directory for %s %q: %q", label, targetPath, write.StagedPath)
		}
		return atomicWriteArgs{
			path:     targetPath,
			label:    label,
			isDir:    false,
			tempPath: write.StagedPath,
		}, nil
	}

	tempFile, err := createManagedTempFile(targetPath)
	if err != nil {
		return atomicWriteArgs{}, fmt.Errorf("failed to create temp file for %s %q: %w", label, targetPath, err)
	}

	// For subsequent operations, ensure the temp file is cleaned up to avoid littering the filesystem with orphaned temp files.
	failWithCleanup := func(format string, innerErr error) (atomicWriteArgs, error) {
		closeAndRemoveTempFile(tempFile)
		return atomicWriteArgs{}, fmt.Errorf(format, label, targetPath, innerErr)
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
		return atomicWriteArgs{}, fmt.Errorf("failed to close temp file for %s %q: %w", label, targetPath, err)
	}

	return atomicWriteArgs{
		path:     targetPath,
		label:    label,
		isDir:    false,
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

// recoverAtomicBackup checks for the existence of the target path and its backup, and attempts to restore from backup if the target is missing.
func recoverAtomicBackup(path string, label string, isDir bool) error {
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
		_ = os.RemoveAll(backupPath)
	}

	writeType := writeTypeLabel(isDir)
	if targetPathErr != nil && !errors.Is(targetPathErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect %s %q for backup recovery (write_type=%s): %w", label, path, writeType, targetPathErr)
	}
	if backupPathErr != nil && !errors.Is(backupPathErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect backup for %s %q (write_type=%s): %w", label, path, writeType, backupPathErr)
	}
	return nil
}

// commitPreparedWrite replaces the target file with the prepared temp file, keeping a backup of the original if it exists.
func commitPreparedWrite(write *atomicWriteArgs) error {
	if info, err := os.Stat(write.path); err == nil {
		if !write.isDir && info.IsDir() {
			return fmt.Errorf("%s target %q is a directory", write.label, write.path)
		}
		if write.isDir && !info.IsDir() {
			return fmt.Errorf("%s target %q is not a directory", write.label, write.path)
		}
		write.hasOriginal = true
		write.backupPath = write.path + ".bak"
		_ = os.RemoveAll(write.backupPath)
		if err := os.Rename(write.path, write.backupPath); err != nil {
			return fmt.Errorf("failed to backup %s %q (write_type=%s): %w", write.label, write.path, writeTypeLabel(write.isDir), err)
		}
		write.backupExists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect %s %q before commit (write_type=%s): %w", write.label, write.path, writeTypeLabel(write.isDir), err)
	}

	if err := os.Rename(write.tempPath, write.path); err != nil {
		if write.backupExists {
			_ = os.Rename(write.backupPath, write.path)
			write.backupExists = false
		}
		return fmt.Errorf("failed to replace %s %q atomically (write_type=%s): %w", write.label, write.path, writeTypeLabel(write.isDir), err)
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
			_ = os.RemoveAll(write.path)
			_ = os.Rename(write.backupPath, write.path)
			continue
		}

		_ = os.RemoveAll(write.path)
	}
}

// cleanupPrepared removes any temp files created during the preparation phase of an atomic write batch.
func cleanupPrepared(prepared []atomicWriteArgs) {
	for _, write := range prepared {
		if write.tempPath != "" {
			_ = os.RemoveAll(write.tempPath)
		}
	}
}

// cleanupBackups removes any backup files created during the commit phase of an atomic write batch.
func cleanupBackups(prepared []atomicWriteArgs) {
	for _, write := range prepared {
		if !write.backupExists || write.backupPath == "" {
			continue
		}
		_ = os.RemoveAll(write.backupPath)
	}
}
