package observability

import (
	"context"
	"time"
	"unicode"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DefaultHistogramBuckets are the standard latency buckets for datasource stats histograms.
// Values in seconds: 50ms to 10s, suitable for most database operations.

const microsecondsPerSecond = 1e6

// titleCase returns the string with the first letter capitalized.
func titleCase(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])

	return string(r)
}

// Observable is implemented by datasources that support instrumentation.
// Datasources embedding instrumentation automatically implement this interface.
type Observable interface {
	SetLogger(Logger)
	SetMetrics(Metrics)
	SetTracer(trace.Tracer)
}

// ObservableQuery is implemented by datasource QueryLog structs.
// It allows OperationStats to set the duration before logging.
type ObservableQuery interface {
	SetDuration(d int64)
	GetOperation() string
	GetMetricLabels() []string
	GetTraceLabels() map[string]string
}

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
//	    instrumentation common.instrumentation
//	    // other fields...
//	}
//
//	func New(c Config) *Client {
//	    return &Client{
//	        instrumentation: common.NewInstrumentation("mongo"),
//	    }
//	}
type instrumentation struct {
	datasourceName string // e.g., "mongo", "arangodb" - used for metric name derivation
	Logger         Logger
	Metrics        Metrics
	Tracer         trace.Tracer
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
		Logger:         NewNoopLogger(),
		Metrics:        NewNoopMetrics(),
	}
}

// SetLogger sets the logger (implements Observable).
func (i *instrumentation) SetLogger(l Logger) {
	if l != nil {
		i.Logger = l
	}
}

// SetMetrics sets the metrics (implements Observable).
func (i *instrumentation) SetMetrics(m Metrics) {
	if m != nil {
		i.Metrics = m
	}
}

// SetTracer sets the tracer (implements Observable).
func (i *instrumentation) SetTracer(t trace.Tracer) {
	if t != nil {
		i.Tracer = t
	}
}

// Logger convenience methods - these delegate to i.Logger for cleaner code.
// Example: c.instrumentation.Logf("...") instead of c.instrumentation.Logger.Logf("...")

// Debug logs at debug level.
func (i *instrumentation) Debug(args ...any) { i.Logger.Debug(args...) }

// Debugf logs at debug level with formatting.
func (i *instrumentation) Debugf(format string, args ...any) { i.Logger.Debugf(format, args...) }

// Info logs at info level.
func (i *instrumentation) Info(args ...any) { i.Logger.Info(args...) }

// Infof logs at info level with formatting.
func (i *instrumentation) Infof(format string, args ...any) { i.Logger.Infof(format, args...) }

// Logf logs at info level with formatting (alias for Infof).
func (i *instrumentation) Logf(format string, args ...any) { i.Logger.Logf(format, args...) }

// Error logs at error level.
func (i *instrumentation) Error(args ...any) { i.Logger.Error(args...) }

// Errorf logs at error level with formatting.
func (i *instrumentation) Errorf(format string, args ...any) { i.Logger.Errorf(format, args...) }

// Warn logs at warn level.
func (i *instrumentation) Warn(args ...any) { i.Logger.Warn(args...) }

// Warnf logs at warn level with formatting.
func (i *instrumentation) Warnf(format string, args ...any) { i.Logger.Warnf(format, args...) }

// Metrics convenience methods - these delegate to i.Metrics for cleaner code.

// NewHistogram creates a new histogram metric.
func (i *instrumentation) NewHistogram(name, desc string, buckets ...float64) {
	i.Metrics.NewHistogram(name, desc, buckets...)
}

// RecordHistogram records a value in a histogram metric.
func (i *instrumentation) RecordHistogram(ctx context.Context, name string, value float64, labels ...string) {
	i.Metrics.RecordHistogram(ctx, name, value, labels...)
}

// NewGauge creates a new gauge metric.
func (i *instrumentation) NewGauge(name, desc string) {
	i.Metrics.NewGauge(name, desc)
}

// SetGauge sets the value of a gauge metric.
func (i *instrumentation) SetGauge(name string, value float64, labels ...string) {
	i.Metrics.SetGauge(name, value, labels...)
}

// NewCounter creates a new counter metric.
func (i *instrumentation) NewCounter(name, desc string) {
	i.Metrics.NewCounter(name, desc)
}

// IncrementCounter increments a counter metric.
func (i *instrumentation) IncrementCounter(ctx context.Context, name string, labels ...string) {
	i.Metrics.IncrementCounter(ctx, name, labels...)
}

// StatsHistogramName returns the standard histogram name for this datasource.
// Format: app_{datasourceName}_stats (e.g., "app_mongo_stats").
func (i *instrumentation) StatsHistogramName() string {
	return "app_" + i.datasourceName + "_stats"
}

// RegisterStatsHistogram registers the standard stats histogram for this datasource.
// The histogram name and description are auto-derived from datasourceName:
//   - Name: app_{datasourceName}_stats
//   - Description: Response time of {DatasourceName} operations in seconds.
//
// Parameters:
//   - buckets: Histogram bucket boundaries (use DefaultHistogramBuckets for standard buckets)
//
// Call this once during Connect() after metrics have been set.
func (i *instrumentation) RegisterStatsHistogram(buckets ...float64) {
	description := "Response time of " + titleCase(i.datasourceName) + " operations in seconds."
	i.Metrics.NewHistogram(i.StatsHistogramName(), description, buckets...)
}

// StartTrace creates a new span for tracing an operation.
// Returns the context with the span and the span itself.
// If tracer is nil, returns the original context and nil span.
//
// Parameters:
//   - ctx: Context for the trace
//   - spanName: Name of the span (e.g., "mongodb-insert", "arangodb-query")
//
// Example:
//
//	ctx, span := c.instrumentation.StartTrace(ctx, "mongodb-insert")
//	if span != nil {
//	    span.SetAttributes(attribute.String("collection", "users"))
//	}
func (i *instrumentation) StartTrace(ctx context.Context, spanName string) (context.Context, trace.Span) {
	if i.Tracer != nil {
		return i.Tracer.Start(ctx, spanName)
	}

	return ctx, nil
}

// AddTrace creates a new span for tracing a datasource operation.
// Span name and attributes are auto-derived from datasourceName:
//   - Span name: {datasourceName}-{operation} (e.g., "mongo-insertOne", "arango-query")
//   - Attributes: prefixed with {datasourceName}. (e.g., "mongo.collection", "arango.DB")
//
// Parameters:
//   - ctx: Context for the trace
//   - query: ObservableQuery - this is used to extract the information about operation and labels
//
// Returns the context with the span and the span itself.
// If tracer is nil, returns the original context and nil span.
func (i *instrumentation) AddTrace(ctx context.Context, query ObservableQuery) (context.Context, trace.Span) {
	if i.Tracer == nil {
		return ctx, nil
	}

	tracerCtx, span := i.Tracer.Start(ctx, i.spanName(query.GetOperation()))

	span.SetAttributes(i.getTraceLabelsForDB(query.GetTraceLabels())...)

	return tracerCtx, span
}

// spanName returns the standard span name for the operation
// Format: {datasourceName}-{operation} (e.g., "monogo-insert").
func (i *instrumentation) spanName(operation string) string {
	return i.datasourceName + "-" + operation
}

// spanDurationKey returns the span attribute key for operation duration.
// Format: {datasourceName}.{method}.duration (e.g., "mongo.insert.duration").
func (i *instrumentation) spanDurationKey(method string) string {
	return i.datasourceName + "." + method + ".duration"
}

// SendOperationStats logs the query, records performance metrics, and ends the span for a datasource operation.
// It calculates duration from startTime and sets it on the log via SetDuration.
// Metric names are automatically derived from the datasourceName set in NewInstrumentation:
//   - Histogram: app_{datasourceName}_stats
//   - Span attribute: {datasourceName}.{method}.duration
//
// Metric Labels are automatically derived from ObservableQuery
//
//
// Parameters:
//   - ctx: Context for metrics recording
//   - log: implementing ObservableQuery interface
//   - startTime: When the operation started (from time.Now() at defer setup)
//   - span: OpenTelemetry span to end (can be nil)

func (i *instrumentation) OperationStats(ctx context.Context, query ObservableQuery,
	startTime time.Time, span trace.Span) {
	duration := time.Since(startTime).Microseconds()
	query.SetDuration(duration)

	i.Logger.Debug(query)

	// Convert microseconds to seconds for histogram buckets
	i.Metrics.RecordHistogram(ctx, i.StatsHistogramName(), float64(duration)/microsecondsPerSecond, query.GetMetricLabels()...)

	if span != nil {
		span.SetAttributes(attribute.Int64(i.spanDurationKey(query.GetOperation()), duration))
		span.End()
	}
}

func (i *instrumentation) getTraceLabelsForDB(labels map[string]string) []attribute.KeyValue {
	traceLabels := make([]attribute.KeyValue, len(labels))

	cnt := 0
	for k, v := range labels {
		traceLabels[cnt] = attribute.String(i.datasourceName+"."+k, v)
		cnt++
	}

	return traceLabels
}

func GetDefaultHistogramBuckets() []float64 {
	return []float64{.05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
}
