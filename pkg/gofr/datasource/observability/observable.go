package observability

import "go.opentelemetry.io/otel/trace"

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
