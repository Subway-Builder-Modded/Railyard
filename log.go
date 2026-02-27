package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Routine string

const (
	StartupRoutine  Routine = "Startup"
	RuntimeRoutine  Routine = "Runtime"
	ShutdownRoutine Routine = "Shutdown"
)

type Logger interface {
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, err error, attrs ...any)
}

// Global logger defaults
const (
	flushInterval = 5 * time.Second
	maxBufferSize = 1 << 20 // 1 MiB
)

type AppLogger struct {
	path string

	mu      sync.Mutex
	buffer  []byte
	dropped int
	routine Routine
	started bool
	stopped bool

	stopCh chan struct{}
	doneCh chan struct{}

	base *slog.Logger
}

// NewMockLogger creates a mock logger for testing purposes
func NewMockLogger() *AppLogger {
	return NewAppLogger()
}

// NewAppLogger creates a new application-level logger that writes to the default log file path
func NewAppLogger() *AppLogger {
	l := &AppLogger{
		path:    LogFilePath(),
		routine: StartupRoutine,
	}

	l.base = slog.New(slog.NewTextHandler(&appLogWriter{owner: l}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return l
}

// Start initializes the logger's background flush routine. Must be called before any log writes will be persisted to disk.
func (l *AppLogger) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.started {
		return nil
	}

	l.started = true
	l.stopped = false
	l.stopCh = make(chan struct{})
	l.doneCh = make(chan struct{})
	interval := flushInterval

	go l.runFlusher(interval, l.stopCh, l.doneCh)
	return nil
}

// Shutdown stops the logger's background flush routine and flushes any remaining logs to disk. 
// Called on application shutdown to ensure all logs are persisted.
func (l *AppLogger) Shutdown() error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil
	}

	stopCh := l.stopCh
	doneCh := l.doneCh
	l.stopCh = nil
	l.doneCh = nil
	l.started = false
	l.stopped = true
	l.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}

	return l.flush()
}

// SetRoutine sets the current routine for log entries. 
// This allows different parts of the application (e.g. startup, runtime, shutdown) to be distinguished in the logs.
func (l *AppLogger) SetRoutine(lc Routine) {
	l.mu.Lock()
	l.routine = lc
	l.mu.Unlock()
}

func (l *AppLogger) forRoutine(lc Routine) *slog.Logger {
	return l.base.With("Routine", string(lc))
}

func (l *AppLogger) logger() *slog.Logger {
	return l.forRoutine(l.currentLifecycle())
}

func (l *AppLogger) Info(msg string, attrs ...any) {
	l.logger().Info(msg, attrs...)
}

func (l *AppLogger) Warn(msg string, attrs ...any) {
	l.logger().Warn(msg, attrs...)
}

func (l *AppLogger) Error(msg string, err error, attrs ...any) {
	if err != nil {
		attrs = append([]any{"Error", err}, attrs...)
	}
	l.logger().Error(msg, attrs...)
}

func (l *AppLogger) runFlusher(interval time.Duration, stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = l.flush()
		case <-stopCh:
			return
		}
	}
}

func (l *AppLogger) flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If buffer is empty and no drops, skip file I/O
	if len(l.buffer) == 0 && l.dropped == 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", l.path, err)
	}
	defer f.Close()

	if l.dropped > 0 {
		if _, err := fmt.Fprintf(f, "time=%q level=WARN msg=%q lifecycle=%q dropped=%d\n",
			time.Now().Format(time.RFC3339Nano),
			"log buffer overflow; dropped oldest bytes",
			l.routine,
			l.dropped,
		); err != nil {
			return fmt.Errorf("failed to write log drop notice: %w", err)
		}
	}

	if _, err := f.Write(l.buffer); err != nil {
		return fmt.Errorf("failed to write log buffer: %w", err)
	}

	l.buffer = l.buffer[:0]
	l.dropped = 0
	return nil
}

func (l *AppLogger) currentLifecycle() Routine {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.routine
}

func (l *AppLogger) appendRaw(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If incoming data exceeds max buffer size, drop buffer and keep tail of incoming data
	rawOverflow := len(p) - maxBufferSize
	combinedOverflow := len(l.buffer) + rawOverflow
	if rawOverflow >= 0 {
		l.buffer = append(l.buffer[:0], p[rawOverflow:]...)
		l.dropped++
		return len(p), nil
		// If combined buffer would exceed max size, drop oldest bytes from buffer
	} else if combinedOverflow > 0 {
		l.buffer = l.buffer[combinedOverflow:]
		l.dropped++
	}
	l.buffer = append(l.buffer, p...)
	return len(p), nil
}

type appLogWriter struct {
	owner *AppLogger
}

func (w *appLogWriter) Write(p []byte) (int, error) {
	return w.owner.appendRaw(p)
}
