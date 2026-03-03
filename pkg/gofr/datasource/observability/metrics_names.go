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

// OperationLabels contains pre-defined labels for datasource operation metrics.
// All fields are optional - empty strings are omitted from metrics.
type OperationLabels struct {
	Host      string // Connection target (hostname/endpoint)
	Database  string // Database name
	operation string // Operation type (query, insert, get, etc.)
	Table     string // Table/collection name
}

// toLabels converts to label key-value pairs for histogram recording.
// Only non-empty values are included.
func (ol OperationLabels) toLabels() []string {
	var labels []string
	if ol.Host != "" {
		labels = append(labels, LabelHost, ol.Host)
	}

	if ol.Database != "" {
		labels = append(labels, LabelDatabase, ol.Database)
	}

	if ol.operation != "" {
		labels = append(labels, LabelOperation, ol.operation)
	}

	if ol.Table != "" {
		labels = append(labels, LabelTable, ol.Table)
	}

	return labels
}

const (

	// Database datasources.
	MongoStatsHistogram         = "app_mongo_stats"
	ArangoStatsHistogram        = "app_arango_stats"
	CassandraStatsHistogram     = "app_cassandra_stats"
	ClickhouseStatsHistogram    = "app_clickhouse_stats"
	DgraphStatsHistogram        = "app_dgraph_stats"
	ElasticsearchStatsHistogram = "app_elasticsearch_stats"
	OracleStatsHistogram        = "app_oracle_stats"
	ScyllaDBStatsHistogram      = "app_scylladb_stats"
	CouchbaseStatsHistogram     = "app_couchbase_stats"
	InfluxDBStatsHistogram      = "app_influxdb_stats"
	OpenTSDBStatsHistogram      = "app_opentsdb_stats"
	SolrStatsHistogram          = "app_solr_stats"
	SurrealDBStatsHistogram     = "app_surrealdb_stats"

	// KV-store datasources.
	DynamoDBStatsHistogram = "app_dynamodb_stats"
	NATSKVStatsHistogram   = "app_nats_kv_stats"
	BadgerStatsHistogram   = "app_badger_stats"
	RedisStatsHistogram    = "app_redis_stats"

	// File storage datasources.
	FTPStatsHistogram   = "app_ftp_stats"
	SFTPStatsHistogram  = "app_sftp_stats"
	S3StatsHistogram    = "app_s3_stats"
	GCSStatsHistogram   = "app_gcs_stats"
	AzureStatsHistogram = "app_azure_stats"
)
