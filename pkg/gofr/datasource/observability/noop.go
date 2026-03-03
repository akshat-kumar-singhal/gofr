package observability

import "context"

// noopLogger discards all log output (safe default).
type noopLogger struct{}

// NewNoopLogger returns a Logger that discards all output.
func NewNoopLogger() Logger {
	return noopLogger{}
}

func (noopLogger) Debug(_ ...any)            {}
func (noopLogger) Debugf(_ string, _ ...any) {}
func (noopLogger) Logf(_ string, _ ...any)   {}
func (noopLogger) Errorf(_ string, _ ...any) {}

// noopMetrics discards all metrics (safe default).
type noopMetrics struct{}

// NewNoopMetrics returns a Metrics that discards all output.
func NewNoopMetrics() Metrics {
	return noopMetrics{}
}

func (noopMetrics) NewHistogram(_, _ string, _ ...float64) {}
func (noopMetrics) RecordHistogram(_ context.Context, _ string, _ float64, _ ...string) {
}
