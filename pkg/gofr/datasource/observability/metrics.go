package observability

import "context"

const microsecondsPerSecond = 1e6

// Metric name constants for datasources.
// All durations are recorded in seconds for histogram consistency.
const (
	// Standard metric label keys for consistency across datasources.
	// Use these instead of creating custom label names.
	LabelHost      = "host"      // Connection target (hostname/endpoint)
	LabelDatabase  = "database"  // Database name
	LabelOperation = "operation" // Operation type (query, insert, get, etc.)
	LabelTable     = "table"     // Table/collection name
	LabelBucket    = "bucket"    // Bucket name (for KV stores)
)

// Metrics is the common metrics interface for all datasources.
// It combines methods from all existing datasource Metrics interfaces,
// providing a unified API for metrics collection.
type Metrics interface {
	NewHistogram(name, desc string, buckets ...float64)
	RecordHistogram(ctx context.Context, name string, value float64, labels ...string)
}

// GetDefaultHistogramBuckets returns the standard latency buckets for datasource stats histograms.
// Values in seconds: 50ms to 10s, suitable for most database operations.
func GetDefaultHistogramBuckets() []float64 {
	return []float64{.05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
}
