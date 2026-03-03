package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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
	if i.tracer != nil {
		return i.tracer.Start(ctx, spanName)
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
	if i.tracer == nil {
		return ctx, nil
	}

	tracerCtx, span := i.tracer.Start(ctx, i.spanName(query.GetOperation()))

	span.SetAttributes(i.getTraceLabelsForDB(query.GetTraceLabels())...)

	return tracerCtx, span
}

// spanName returns the standard span name for the operation
// Format: {datasourceName}-{operation} (e.g., "mongo-insert").
func (i *instrumentation) spanName(operation string) string {
	return i.datasourceName + "-" + operation
}

// spanDurationKey returns the span attribute key for operation duration.
// Format: {datasourceName}.{method}.duration (e.g., "mongo.insert.duration").
func (i *instrumentation) spanDurationKey(method string) string {
	return i.datasourceName + "." + method + ".duration"
}

// getTraceLabelsForDB converts a map of trace labels to OpenTelemetry attributes prefixed with the datasource name.
func (i *instrumentation) getTraceLabelsForDB(labels map[string]string) []attribute.KeyValue {
	traceLabels := make([]attribute.KeyValue, len(labels))

	cnt := 0
	for k, v := range labels {
		traceLabels[cnt] = attribute.String(i.datasourceName+"."+k, v)
		cnt++
	}

	return traceLabels
}
