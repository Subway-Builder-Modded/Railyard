package testutil

import "railyard/internal/types"

type TestLogSink struct{}

func (TestLogSink) Info(string, ...any)                               {}
func (TestLogSink) Warn(string, ...any)                               {}
func (TestLogSink) Error(string, error, ...any)                       {}
func (TestLogSink) MultipleError(string, []error, ...any)             {}
func (TestLogSink) LogResponse(string, types.GenericResponse, ...any) {}
