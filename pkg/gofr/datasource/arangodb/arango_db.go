package arangodb

import (
	"context"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"

	"gofr.dev/pkg/gofr/datasource/observability"
)

type DB struct {
	client *Client
}

// CreateDB creates a new database in ArangoDB.
// It first checks if the database already exists before attempting to create it.
// Returns ErrDatabaseExists if the database already exists.
func (d *DB) CreateDB(ctx context.Context, database string) error {
	tracerCtx, span := d.client.instrumentation.AddTrace(ctx, "createDB", map[string]string{
		"DB": database,
	})

	defer d.client.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "createDB", Database: database},
		time.Now(), "createDB", span,
		observability.OperationLabels{Host: d.client.endpoint})

	// Check if the database already exists
	exists, err := d.client.client.DatabaseExists(tracerCtx, database)
	if err != nil {
		return err
	}

	if exists {
		d.client.instrumentation.Debugf("database %s already exists", database)
		return ErrDatabaseExists
	}

	_, err = d.client.client.CreateDatabase(tracerCtx, database, nil)

	return err
}

// DropDB deletes a database from ArangoDB.
func (d *DB) DropDB(ctx context.Context, database string) error {
	tracerCtx, span := d.client.instrumentation.AddTrace(ctx, "dropDB", map[string]string{
		"DB": database,
	})

	defer d.client.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "dropDB", Database: database},
		time.Now(), "dropDB", span,
		observability.OperationLabels{Host: d.client.endpoint})

	db, err := d.client.client.GetDatabase(tracerCtx, database, &arangodb.GetDatabaseOptions{})
	if err != nil {
		return err
	}

	err = db.Remove(tracerCtx)
	if err != nil {
		return err
	}

	return err
}

// CreateCollection creates a new collection in a database with specified type.
// It first checks if the collection already exists before attempting to create it.
// Returns ErrCollectionExists if the collection already exists.
func (d *DB) CreateCollection(ctx context.Context, database, collection string, isEdge bool) error {
	tracerCtx, span := d.client.instrumentation.AddTrace(ctx, "createCollection", map[string]string{
		"collection": collection,
	})

	defer d.client.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "createCollection", Database: database, Collection: collection, Filter: isEdge},
		time.Now(), "createCollection", span,
		observability.OperationLabels{Host: d.client.endpoint, Table: collection})

	db, err := d.client.client.GetDatabase(tracerCtx, database, nil)
	if err != nil {
		return err
	}

	// Check if the collection already exists
	exists, err := db.CollectionExists(tracerCtx, collection)
	if err != nil {
		return err
	}

	if exists {
		d.client.instrumentation.Debugf("collection %s already exists in database %s", collection, database)
		return ErrCollectionExists
	}

	collectionType := arangodb.CollectionTypeDocument
	if isEdge {
		collectionType = arangodb.CollectionTypeEdge
	}

	options := arangodb.CreateCollectionPropertiesV2{Type: &collectionType}

	_, err = db.CreateCollectionV2(tracerCtx, collection, &options)

	return err
}

// DropCollection deletes an existing collection from a database.
func (d *DB) DropCollection(ctx context.Context, database, collectionName string) error {
	return d.handleCollectionOperation(ctx, "dropCollection", database, collectionName, func(collection arangodb.Collection) error {
		return collection.Remove(ctx)
	})
}

func (d *DB) getCollection(ctx context.Context, dbName, collectionName string) (arangodb.Collection, error) {
	db, err := d.client.client.GetDatabase(ctx, dbName, nil)
	if err != nil {
		return nil, err
	}

	collection, err := db.GetCollection(ctx, collectionName, nil)
	if err != nil {
		return nil, err
	}

	return collection, nil
}

// handleCollectionOperation handles common logic for collection operations.
func (d *DB) handleCollectionOperation(ctx context.Context, operation, database, collectionName string,
	action func(arangodb.Collection) error) error {
	tracerCtx, span := d.client.instrumentation.AddTrace(ctx, operation, map[string]string{
		"collection": collectionName,
	})

	defer d.client.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: operation, Database: database, Collection: collectionName},
		time.Now(), operation, span,
		observability.OperationLabels{Host: d.client.endpoint, Table: collectionName})

	collection, err := d.getCollection(tracerCtx, database, collectionName)
	if err != nil {
		return err
	}

	return action(collection)
}
