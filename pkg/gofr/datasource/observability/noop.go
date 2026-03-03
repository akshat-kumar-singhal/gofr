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
func (noopLogger) Info(_ ...any)             {}
func (noopLogger) Infof(_ string, _ ...any)  {}
func (noopLogger) Logf(_ string, _ ...any)   {}
func (noopLogger) Error(_ ...any)            {}
func (noopLogger) Errorf(_ string, _ ...any) {}
func (noopLogger) Warn(_ ...any)             {}
func (noopLogger) Warnf(_ string, _ ...any)  {}

// noopMetrics discards all metrics (safe default).
type noopMetrics struct{}

// NewNoopMetrics returns a Metrics that discards all output.
func NewNoopMetrics() Metrics {
	return noopMetrics{}
}

func (noopMetrics) NewHistogram(_, _ string, _ ...float64) {}
func (noopMetrics) RecordHistogram(_ context.Context, _ string, _ float64, _ ...string) {
}
func (noopMetrics) NewGauge(_, _ string)                                      {}
func (noopMetrics) SetGauge(_ string, _ float64, _ ...string)                 {}
func (noopMetrics) NewCounter(_, _ string)                                    {}
func (noopMetrics) IncrementCounter(_ context.Context, _ string, _ ...string) {}
