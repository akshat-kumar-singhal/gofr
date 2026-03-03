package mongo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel/trace"

	"gofr.dev/pkg/gofr/datasource/observability"
)

type Client struct {
	*mongo.Database

	uri             string
	database        string
	instrumentation observability.Instrumenter
	config          *Config
}

type Config struct {
	Host     string
	User     string
	Password string
	Port     int
	Database string
	// Deprecated Provide Host User Password Port Instead and driver will generate the URI
	URI               string
	ConnectionTimeout time.Duration
}

const defaultTimeout = 5 * time.Second

var (
	errStatusDown   = errors.New("status down")
	errMissingField = errors.New("missing required field in config")
	errIncorrectURI = errors.New("incorrect URI for MongoDB")
	errParseHost    = errors.New("failed to parse host from MongoDB URI")
)

/*
Developer Note: We could have accepted logger and metrics as part of the factory function `New`, but when mongo driver is
initialized in GoFr, We want to ensure that the user need not to provides logger and metrics and then connect to the database,
i.e. by default observability features gets initialized when used with GoFr.
*/

// New initializes MongoDB driver with the provided configuration.
// The Connect method must be called to establish a connection to MongoDB.
// Usage:
// client := New(config)
// client.SetLogger(loggerInstance)
// client.SetMetrics(metricsInstance)
// client.Connect().
//
//nolint:gocritic // Configs do not need to be passed by reference
func New(c Config) *Client {
	return &Client{
		instrumentation: observability.NewInstrumentation("mongo"),
		config:          &c,
	}
}

// SetLogger sets the logger for the MongoDB client.
func (c *Client) SetLogger(l observability.Logger) {
	c.instrumentation.SetLogger(l)
}

// SetMetrics sets the metrics for the MongoDB client.
func (c *Client) SetMetrics(m observability.Metrics) {
	c.instrumentation.SetMetrics(m)
}

// SetTracer sets the tracer for the MongoDB client.
func (c *Client) SetTracer(t trace.Tracer) {
	c.instrumentation.SetTracer(t)
}

// UseLogger sets the logger for the MongoDB client which asserts the Logger interface.
//
// Deprecated: Use SetLogger instead.
func (c *Client) UseLogger(logger any) {
	if l, ok := logger.(observability.Logger); ok {
		c.SetLogger(l)
	}
}

// UseMetrics sets the metrics for the MongoDB client which asserts the Metrics interface.
//
// Deprecated: Use SetMetrics instead.
func (c *Client) UseMetrics(metrics any) {
	if m, ok := metrics.(observability.Metrics); ok {
		c.SetMetrics(m)
	}
}

// UseTracer sets the tracer for the MongoDB client.
//
// Deprecated: Use SetTracer instead.
func (c *Client) UseTracer(tracer any) {
	if t, ok := tracer.(trace.Tracer); ok {
		c.SetTracer(t)
	}
}

// Connect establishes a connection to MongoDB and registers metrics using the provided configuration when the client was Created.
func (c *Client) Connect() {
	uri, host, err := generateMongoURI(c.config)
	if err != nil {
		c.instrumentation.Errorf("error generating MongoDB URI: %v", err)
		return
	}

	c.instrumentation.Debugf("connecting to MongoDB at %v to database %v", c.config.Host, c.config.Database)

	timeout := c.config.ConnectionTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	m, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		c.instrumentation.Errorf("error while connecting to MongoDB, err:%v", err)

		return
	}

	if err = m.Ping(ctx, nil); err != nil {
		c.instrumentation.Errorf("could not connect to MongoDB at %v due to err: %v", host, err)
		return
	}

	// Register standard stats histogram (auto-derives name and description from datasource name)
	c.instrumentation.RegisterStatsHistogram(observability.DefaultHistogramBuckets...)

	c.Database = m.Database(c.config.Database)
	c.uri = uri
	c.database = c.config.Database

	c.instrumentation.Logf("connected to MongoDB at %v to database %v", host, c.config.Database)
}

func generateMongoURI(config *Config) (uri, host string, err error) {
	if config.URI != "" {
		host, err = getDBHost(config.URI)
		if err != nil {
			return "", "", err
		}

		return config.URI, host, nil
	}

	switch {
	case config.Host == "":
		return "", "", fmt.Errorf("%w: host is empty", errMissingField)
	case config.Port == 0:
		return "", "", fmt.Errorf("%w: port is empty", errMissingField)
	case config.Database == "":
		return "", "", fmt.Errorf("%w: database is empty", errMissingField)
	}

	u := &url.URL{
		Scheme: "mongodb",
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Path:   "/" + url.PathEscape(config.Database),
	}

	if config.User != "" && config.Password != "" {
		u.User = url.UserPassword(url.QueryEscape(config.User), url.QueryEscape(config.Password))
	}

	q := u.Query()
	q.Set("authSource", "admin")
	u.RawQuery = q.Encode()

	return u.String(), u.Hostname(), nil
}

func getDBHost(uri string) (host string, err error) {
	parsedURL, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme != "mongodb" {
		return "", errIncorrectURI
	}

	if parsedURL.Hostname() == "" {
		return "", errParseHost
	}

	return parsedURL.Hostname(), nil
}

// InsertOne inserts a single document into the specified collection.
func (c *Client) InsertOne(ctx context.Context, collection string, document any) (any, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "insertOne", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "insertOne", Collection: collection, Filter: document},
		time.Now(), "insert", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	result, err := c.Database.Collection(collection).InsertOne(tracerCtx, document)

	return result, err
}

// InsertMany inserts multiple documents into the specified collection.
func (c *Client) InsertMany(ctx context.Context, collection string, documents []any) ([]any, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "insertMany", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "insertMany", Collection: collection, Filter: documents},
		time.Now(), "insertMany", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	res, err := c.Database.Collection(collection).InsertMany(tracerCtx, documents)
	if err != nil {
		return nil, err
	}

	return res.InsertedIDs, nil
}

// Find retrieves documents from the specified collection based on the provided filter and binds response to result.
func (c *Client) Find(ctx context.Context, collection string, filter, results any) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "find", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "find", Collection: collection, Filter: filter},
		time.Now(), "find", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	cur, err := c.Database.Collection(collection).Find(tracerCtx, filter)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)

	if err := cur.All(ctx, results); err != nil {
		return err
	}

	return nil
}

// FindOne retrieves a single document from the specified collection based on the provided filter and binds response to result.
func (c *Client) FindOne(ctx context.Context, collection string, filter, result any) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "findOne", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "findOne", Collection: collection, Filter: filter},
		time.Now(), "findOne", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	b, err := c.Database.Collection(collection).FindOne(tracerCtx, filter).Raw()
	if err != nil {
		return err
	}

	return bson.Unmarshal(b, result)
}

// UpdateByID updates a document in the specified collection by its ID.
func (c *Client) UpdateByID(ctx context.Context, collection string, id, update any) (int64, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "updateByID", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "updateByID", Collection: collection, ID: id, Update: update},
		time.Now(), "updateByID", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	res, err := c.Database.Collection(collection).UpdateByID(tracerCtx, id, update)

	return res.ModifiedCount, err
}

// UpdateOne updates a single document in the specified collection based on the provided filter.
func (c *Client) UpdateOne(ctx context.Context, collection string, filter, update any) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "updateOne", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "updateOne", Collection: collection, Filter: filter, Update: update},
		time.Now(), "updateOne", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	_, err := c.Database.Collection(collection).UpdateOne(tracerCtx, filter, update)

	return err
}

// UpdateMany updates multiple documents in the specified collection based on the provided filter.
func (c *Client) UpdateMany(ctx context.Context, collection string, filter, update any) (int64, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "updateMany", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "updateMany", Collection: collection, Filter: filter, Update: update},
		time.Now(), "updateMany", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	res, err := c.Database.Collection(collection).UpdateMany(tracerCtx, filter, update)

	return res.ModifiedCount, err
}

// CountDocuments counts the number of documents in the specified collection based on the provided filter.
func (c *Client) CountDocuments(ctx context.Context, collection string, filter any) (int64, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "countDocuments", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "countDocuments", Collection: collection, Filter: filter},
		time.Now(), "countDocuments", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	result, err := c.Database.Collection(collection).CountDocuments(tracerCtx, filter)

	return result, err
}

// DeleteOne deletes a single document from the specified collection based on the provided filter.
func (c *Client) DeleteOne(ctx context.Context, collection string, filter any) (int64, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "deleteOne", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "deleteOne", Collection: collection, Filter: filter},
		time.Now(), "deleteOne", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	res, err := c.Database.Collection(collection).DeleteOne(tracerCtx, filter)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

// DeleteMany deletes multiple documents from the specified collection based on the provided filter.
func (c *Client) DeleteMany(ctx context.Context, collection string, filter any) (int64, error) {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "deleteMany", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "deleteMany", Collection: collection, Filter: filter},
		time.Now(), "deleteMany", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	res, err := c.Database.Collection(collection).DeleteMany(tracerCtx, filter)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

// Drop drops the specified collection from the database.
func (c *Client) Drop(ctx context.Context, collection string) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "drop", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "drop", Collection: collection},
		time.Now(), "drop", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: collection})

	err := c.Database.Collection(collection).Drop(tracerCtx)

	return err
}

// CreateCollection creates the specified collection in the database.
func (c *Client) CreateCollection(ctx context.Context, name string) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "createCollection", map[string]string{
		"collection": name,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Query: "createCollection", Collection: name},
		time.Now(), "createCollection", span,
		observability.OperationLabels{Host: c.uri, Database: c.database, Table: name})

	err := c.Database.CreateCollection(tracerCtx, name)

	return err
}

type Health struct {
	Status  string         `json:"status,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// HealthCheck checks the health of the MongoDB client by pinging the database.
func (c *Client) HealthCheck(ctx context.Context) (any, error) {
	h := Health{
		Details: make(map[string]any),
	}

	h.Details["host"] = c.uri
	h.Details["database"] = c.database

	err := c.Database.Client().Ping(ctx, readpref.Primary())
	if err != nil {
		h.Status = "DOWN"

		return &h, errStatusDown
	}

	h.Status = "UP"

	return &h, nil
}

func (c *Client) StartSession() (any, error) {
	defer c.instrumentation.OperationStats(context.Background(),
		&QueryLog{Query: "startSession"},
		time.Now(), "startSession", nil,
		observability.OperationLabels{Host: c.uri, Database: c.database})

	s, err := c.Client().StartSession()
	ses := &session{s}

	return ses, err
}

type session struct {
	mongo.Session
}

func (s *session) StartTransaction() error {
	return s.Session.StartTransaction()
}

type Transaction interface {
	StartTransaction() error
	AbortTransaction(context.Context) error
	CommitTransaction(context.Context) error
	EndSession(context.Context)
}
