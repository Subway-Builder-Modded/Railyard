package files

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

type JSONReadOptions struct {
	AllowMissing bool
	AllowEmpty   bool
}

// ReadJSON reads a JSON file at path into defined struct type T.
// The label is used for annotating error messages.
// Options on file existence and file content can be set with JSONReadOptions.
func ReadJSON[T any](path string, label string, opts JSONReadOptions) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	// If the file is missing, attempt to recover from backup if allowed, then try reading again. If still missing and AllowMissing is true, return zero value without error.
	if errors.Is(err, fs.ErrNotExist) {
		if recoverErr := recoverAtomicBackup(path, label, false); recoverErr != nil {
			return zero, recoverErr
		}
		data, err = os.ReadFile(path)
	}
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && opts.AllowMissing {
			return zero, nil
		}
		return zero, fmt.Errorf("failed to read %s %q: %w", label, path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		if opts.AllowEmpty {
			return zero, nil
		}
		return zero, fmt.Errorf("failed to parse %s %q: file is empty", label, path)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		return zero, fmt.Errorf("failed to parse %s %q: %w", label, path, err)
	}

	return decoded, nil
}

// ParseJSON parses JSON data into defined struct type T.
// The label is used for annotating error messages.
func ParseJSON[T any](data []byte, label string) (T, error) {
	var zero T
	if len(bytes.TrimSpace(data)) == 0 {
		return zero, fmt.Errorf("failed to parse %s: data is empty", label)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		return zero, fmt.Errorf("failed to parse %s: %w", label, err)
	}
	return decoded, nil
}

// WriteJSON formats the value to JSON and writes it to path.
// The label is used for annotating error messages.
func WriteJSON[T any](path string, label string, value T) error {
	// Format the JSON with indentation for readability
	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize %s: %w", label, err)
	}

	if err := WriteFilesAtomically([]AtomicFileWrite{
		{
			Path:  path,
			Label: label,
			Data:  formatted,
			Perm:  0o644,
		},
	}); err != nil {
		return fmt.Errorf("failed to write %s %q: %w", label, path, err)
	}

	return nil
}
