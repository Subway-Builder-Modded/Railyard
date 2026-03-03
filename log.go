package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"railyard/internal/types"
	"sync"
	"time"
)

type Logger interface {
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, err error, attrs ...any)
	LogResponse(msg string, response types.GenericResponse, attrs ...any)
}

// Global logger defaults
const (
	flushInterval   = 5 * time.Second
	writeBufferSize = 64 * 1024 // 64 KiB
)

type AppLogger struct {
	path string

	mu sync.Mutex

	stopCh chan struct{}
	doneCh chan struct{}

	file   *os.File
	writer *bufio.Writer

	base *slog.Logger
}

// loggerAtPath creates a new logger that writes to the provided file path
// Useful for testing to isolate log output to a known temporary file
func loggerAtPath(path string) *AppLogger {
	if path == "" {
		path = LogFilePath()
	}

	l := &AppLogger{path: path}

	l.base = slog.New(slog.NewTextHandler(&appLogWriter{owner: l}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return l
}

// NewAppLogger creates a new application-level logger that writes to the default log file path.
func NewAppLogger() *AppLogger {
	return loggerAtPath(LogFilePath())
}

// Start initializes the logger's background flush routine. Must be called before any log writes will be persisted to disk.
func (l *AppLogger) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stopCh != nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("Failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("Failed to open log file %q: %w", l.path, err)
	}

	l.file, l.writer = f, bufio.NewWriterSize(f, writeBufferSize)
	l.stopCh, l.doneCh = make(chan struct{}), make(chan struct{})

	// Goroutine to periodically flush log buffer to disk until logger is shutdown
	go func(stopCh <-chan struct{}, doneCh chan<- struct{}) {
		defer close(doneCh)
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = l.flush()
			case <-stopCh:
				return
			}
		}
	}(l.stopCh, l.doneCh)

	return nil
}

// Shutdown stops the logger's background flush routine and flushes any remaining logs to disk.
// Called on application shutdown to ensure all logs are persisted.
func (l *AppLogger) Shutdown() error {
	l.mu.Lock()
	stopCh, doneCh := l.stopCh, l.doneCh
	l.stopCh, l.doneCh = nil, nil
	l.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh // Wait for flush goroutine to exit before closing file
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var flushErr error
	if l.writer != nil {
		if err := l.writer.Flush(); err != nil {
			flushErr = fmt.Errorf("Failed to flush log writer: %w", err)
		}
		l.writer = nil
	}

	var closeErr error
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			closeErr = fmt.Errorf("Failed to close log file: %w", err)
		}
		l.file = nil
	}

	return errors.Join(flushErr, closeErr)
}

func (l *AppLogger) Info(msg string, attrs ...any) {
	l.base.Info(msg, attrs...)
}

func (l *AppLogger) Warn(msg string, attrs ...any) {
	l.base.Warn(msg, attrs...)
}

func (l *AppLogger) Error(msg string, err error, attrs ...any) {
	if err != nil {
		attrs = append([]any{"error", err}, attrs...)
	}
	l.base.Error(msg, attrs...)
}

func (l *AppLogger) LogResponse(msg string, response types.GenericResponse, attrs ...any) {
	baseAttrs := append([]any{"status", response.Status, "message", response.Message}, attrs...)

	switch response.Status {
	case types.ResponseSuccess:
		l.Info(msg, baseAttrs...)
	case types.ResponseWarn:
		l.Warn(msg, baseAttrs...)
	case types.ResponseError:
		l.Error(msg, nil, baseAttrs...)
	default:
		l.Warn(msg, append(baseAttrs, "unknown_status", true)...)
	}
}

func (l *AppLogger) flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer == nil {
		return nil
	}

	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("Failed to flush log writer: %w", err)
	}

	return nil
}

func (l *AppLogger) writeRaw(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer == nil { // Drop log if writer is not initialized
		return len(p), nil
	}

	n, err := l.writer.Write(p)
	if err != nil {
		return n, fmt.Errorf("Failed to write log buffer: %w", err)
	}

	return len(p), nil
}

type appLogWriter struct {
	owner *AppLogger
}

func (w *appLogWriter) Write(p []byte) (int, error) {
	return w.owner.writeRaw(p)
}
