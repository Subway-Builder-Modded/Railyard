package registry

type testLogSink struct{}

func (testLogSink) Info(string, ...any)         {}
func (testLogSink) Warn(string, ...any)         {}
func (testLogSink) Error(string, error, ...any) {}
