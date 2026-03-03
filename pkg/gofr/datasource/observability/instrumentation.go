package observability

import (
	"context"
	"time"
	"unicode"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// titleCase returns the string with the first letter capitalized.
func titleCase(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])

	return string(r)
}

// Instrumenter defines the interface for datasource instrumentation providing logging, metrics, and tracing.
type Instrumenter interface {
	SetLogger(Logger)
	SetMetrics(Metrics)
	SetTracer(trace.Tracer)

	Logf(string, ...any)
	Debug(args ...any)
	Debugf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)

	NewHistogram(string, string, ...float64)
	RecordHistogram(context.Context, string, float64, ...string)
	RegisterStatsHistogram(...float64)

	NewGauge(string, string)
	SetGauge(string, float64, ...string)

	NewCounter(string, string)
	IncrementCounter(context.Context, string, ...string)
	// DecrementCounter(context.Context, string, ...string)

	OperationStats(context.Context, ObservableQuery, time.Time, trace.Span)

	StartTrace(context.Context, string) (context.Context, trace.Span)
	AddTrace(context.Context, ObservableQuery) (context.Context, trace.Span)
}

// instrumentation provides logging, metrics, and tracing.
// Embed this in datasource clients to get SetLogger/SetMetrics/SetTracer methods
// automatically implemented.
//
// Example usage:
//
//	type Client struct {
//	    instrumentation observability.Instrumenter
//	    // other fields...
//	}
//
//	func New(c Config) *Client {
//	    return &Client{
//	        instrumentation: observability.NewInstrumentation("mongo"),
//	    }
//	}
type instrumentation struct {
	datasourceName string // e.g., "mongo", "arangodb" - used for metric name derivation
	logger         Logger
	metrics        Metrics
	tracer         trace.Tracer
}

// NewInstrumentation returns an instrumentation for a specific datasource with safe no-op defaults.
// The datasourceName is used to automatically construct metric names:
//   - Histogram: app_{datasourceName}_stats
//   - Span attribute: {datasourceName}.{method}.duration
//
// This ensures no nil pointer panics if SetLogger/SetMetrics are not called.
func NewInstrumentation(datasourceName string) Instrumenter {
	return &instrumentation{
		datasourceName: datasourceName,
		logger:         NewNoopLogger(),
		metrics:        NewNoopMetrics(),
	}
}

// SetLogger sets the logger (implements Observable).
func (i *instrumentation) SetLogger(l Logger) {
	if l != nil {
		i.logger = l
	}
}

// SetMetrics sets the metrics (implements Observable).
func (i *instrumentation) SetMetrics(m Metrics) {
	if m != nil {
		i.metrics = m
	}
}

// SetTracer sets the tracer (implements Observable).
func (i *instrumentation) SetTracer(t trace.Tracer) {
	if t != nil {
		i.tracer = t
	}
}

// Logger convenience methods - these delegate to i.logger for cleaner code.
// Example: c.instrumentation.Logf("...") instead of c.instrumentation.logger.Logf("...")

// Debug logs at debug level.
func (i *instrumentation) Debug(args ...any) { i.logger.Debug(args...) }

// Debugf logs at debug level with formatting.
func (i *instrumentation) Debugf(format string, args ...any) { i.logger.Debugf(format, args...) }

// Info logs at info level.
func (i *instrumentation) Info(args ...any) { i.logger.Info(args...) }

// Infof logs at info level with formatting.
func (i *instrumentation) Infof(format string, args ...any) { i.logger.Infof(format, args...) }

// Logf logs at info level with formatting (alias for Infof).
func (i *instrumentation) Logf(format string, args ...any) { i.logger.Logf(format, args...) }

// Error logs at error level.
func (i *instrumentation) Error(args ...any) { i.logger.Error(args...) }

// Errorf logs at error level with formatting.
func (i *instrumentation) Errorf(format string, args ...any) { i.logger.Errorf(format, args...) }

// Warn logs at warn level.
func (i *instrumentation) Warn(args ...any) { i.logger.Warn(args...) }

// Warnf logs at warn level with formatting.
func (i *instrumentation) Warnf(format string, args ...any) { i.logger.Warnf(format, args...) }

// Metrics convenience methods - these delegate to i.metrics for cleaner code.

// NewHistogram creates a new histogram metric.
func (i *instrumentation) NewHistogram(name, desc string, buckets ...float64) {
	i.metrics.NewHistogram(name, desc, buckets...)
}

// RecordHistogram records a value in a histogram metric.
func (i *instrumentation) RecordHistogram(ctx context.Context, name string, value float64, labels ...string) {
	i.metrics.RecordHistogram(ctx, name, value, labels...)
}

// NewGauge creates a new gauge metric.
func (i *instrumentation) NewGauge(name, desc string) {
	i.metrics.NewGauge(name, desc)
}

// SetGauge sets the value of a gauge metric.
func (i *instrumentation) SetGauge(name string, value float64, labels ...string) {
	i.metrics.SetGauge(name, value, labels...)
}

// NewCounter creates a new counter metric.
func (i *instrumentation) NewCounter(name, desc string) {
	i.metrics.NewCounter(name, desc)
}

// IncrementCounter increments a counter metric.
func (i *instrumentation) IncrementCounter(ctx context.Context, name string, labels ...string) {
	i.metrics.IncrementCounter(ctx, name, labels...)
}

// statsHistogramName returns the standard histogram name for this datasource.
// Format: app_{datasourceName}_stats (e.g., "app_mongo_stats").
func (i *instrumentation) statsHistogramName() string {
	return "app_" + i.datasourceName + "_stats"
}

// RegisterStatsHistogram registers the standard stats histogram for this datasource.
// The histogram name and description are auto-derived from datasourceName:
//   - Name: app_{datasourceName}_stats
//   - Description: Response time of {DatasourceName} operations in seconds.
//
// Parameters:
//   - buckets: Histogram bucket boundaries (use GetDefaultHistogramBuckets() for standard buckets)
//
// Call this once during Connect() after metrics have been set.
func (i *instrumentation) RegisterStatsHistogram(buckets ...float64) {
	description := "Response time of " + titleCase(i.datasourceName) + " operations in seconds."
	i.metrics.NewHistogram(i.statsHistogramName(), description, buckets...)
}

// OperationStats logs the query, records performance metrics, and ends the span for a datasource operation.
// It calculates duration from startTime and sets it on the log via SetDuration.
// Metric names are automatically derived from the datasourceName set in NewInstrumentation:
//   - Histogram: app_{datasourceName}_stats
//   - Span attribute: {datasourceName}.{method}.duration
//
// # Metric Labels are automatically derived from ObservableQuery
//
// Parameters:
//   - ctx: Context for metrics recording
//   - query: implementing ObservableQuery interface
//   - startTime: When the operation started (from time.Now() at defer setup)
//   - span: OpenTelemetry span to end (can be nil)
func (i *instrumentation) OperationStats(ctx context.Context, query ObservableQuery,
	startTime time.Time, span trace.Span) {
	duration := time.Since(startTime).Microseconds()
	query.SetDuration(duration)

	i.logger.Debug(query)

	// Convert microseconds to seconds for histogram buckets
	i.metrics.RecordHistogram(ctx, i.statsHistogramName(), float64(duration)/microsecondsPerSecond, query.GetMetricLabels()...)

	if span != nil {
		span.SetAttributes(attribute.Int64(i.spanDurationKey(query.GetOperation()), duration))
		span.End()
	}
}
