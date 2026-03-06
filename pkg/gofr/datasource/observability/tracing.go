package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// addTrace creates a new span for tracing a datasource operation.
// Span name and attributes are auto-derived from datasourceName:
//   - Span name: {datasourceName}-{operation} (e.g., "mongo-insertOne", "arango-query")
//   - Attributes: prefixed with {datasourceName}. (e.g., "mongo.collection", "arango.DB")
//
// Returns the context with the span and the span itself.
// If tracer is nil, returns the original context and nil span.
func (i *instrumentation) addTrace(ctx context.Context, query ObservableQuery) (context.Context, trace.Span) {
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
