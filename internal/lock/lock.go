package lock

import (
	"fmt"
	"os"
	"sync"

	"railyard/internal/paths"

	"github.com/gofrs/flock"
)

// Handle is a releasable app-level instance lock.
type Handle interface {
	Release() error
}

// Lock is an OS-backed process lock for single-instance enforcement.
type Lock struct {
	lock     *flock.Flock
	mu       sync.Mutex
	released bool
}

// Acquire acquires the process-wide app lock used to prevent multiple Railyard instances.
func Acquire() (Handle, bool, error) {
	if err := os.MkdirAll(paths.AppDataRoot(), 0o755); err != nil {
		return nil, false, err
	}

	fileLock := flock.New(paths.LockFilePath())
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, false, fmt.Errorf("failed to acquire instance lock: %w", err)
	}
	if !locked {
		return nil, true, nil
	}

	return &Lock{lock: fileLock}, false, nil
}

// Release unlocks the held lock.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.released {
		return nil
	}

	if err := l.lock.Unlock(); err != nil {
		return fmt.Errorf("failed to release instance lock: %w", err)
	}

	l.released = true
	return nil
}
