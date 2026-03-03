package observability

import "context"

// Metrics is the common metrics interface for all datasources.
// It combines methods from all existing datasource Metrics interfaces,
// providing a unified API for metrics collection.
type Metrics interface {
	NewHistogram(name, desc string, buckets ...float64)
	RecordHistogram(ctx context.Context, name string, value float64, labels ...string)
	NewGauge(name, desc string)
	SetGauge(name string, value float64, labels ...string)
	NewCounter(name, desc string)
	IncrementCounter(ctx context.Context, name string, labels ...string)
}
