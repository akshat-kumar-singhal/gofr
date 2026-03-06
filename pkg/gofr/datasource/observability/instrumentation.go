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
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)

	RegisterStatsHistogram(...float64)
	InstrumentOperation(ctx context.Context, op ObservableQuery) (tracerCtx context.Context, end func())
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

// Debugf logs at debug level with formatting.
func (i *instrumentation) Debugf(format string, args ...any) { i.logger.Debugf(format, args...) }

// Logf logs at info level with formatting.
func (i *instrumentation) Logf(format string, args ...any) { i.logger.Logf(format, args...) }

// Errorf logs at error level with formatting.
func (i *instrumentation) Errorf(format string, args ...any) { i.logger.Errorf(format, args...) }

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

// operationStats logs the query, records performance metrics, and ends the span for a datasource operation.
// It calculates duration from startTime and sets it on the log via SetDuration.
// Metric names are automatically derived from the datasourceName set in NewInstrumentation:
//   - Histogram: app_{datasourceName}_stats
//   - Span attribute: {datasourceName}.{method}.duration
func (i *instrumentation) operationStats(query ObservableQuery, startTime time.Time, span trace.Span) {
	duration := time.Since(startTime).Microseconds()
	query.SetDuration(duration)

	i.logger.Debug(query)

	// Convert microseconds to seconds for histogram buckets
	i.metrics.RecordHistogram(context.Background(), i.statsHistogramName(),
		float64(duration)/microsecondsPerSecond, query.GetMetricLabels()...)

	if span != nil {
		span.SetAttributes(attribute.Int64(i.spanDurationKey(query.GetOperation()), duration))
		span.End()
	}
}

// InstrumentOperation starts a trace span and returns the traced context along with a cleanup
// function that records operation stats when called (typically via defer).
func (i *instrumentation) InstrumentOperation(ctx context.Context, op ObservableQuery) (tracerCtx context.Context, end func()) {
	tracerCtx, span := i.addTrace(ctx, op)
	startTime := time.Now()

	return tracerCtx, func() {
		i.operationStats(op, startTime, span)
	}
}
