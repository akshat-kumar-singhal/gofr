package observability

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
