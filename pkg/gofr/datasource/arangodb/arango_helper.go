package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"

	"gofr.dev/pkg/gofr/datasource/observability"
)

func (c *Client) user(ctx context.Context, username string) (arangodb.User, error) {
	return c.client.User(ctx, username)
}

func (c *Client) database(ctx context.Context, name string) (arangodb.Database, error) {
	return c.client.GetDatabase(ctx, name, nil)
}

// createUser creates a new user in ArangoDB.
func (c *Client) createUser(ctx context.Context, username string, options any) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "createUser", map[string]string{
		"user": username,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "createUser", ID: username},
		time.Now(), "createUser", span,
		observability.OperationLabels{Host: c.endpoint})

	userOptions, ok := options.(UserOptions)
	if !ok {
		return fmt.Errorf("%w", errInvalidUserOptionsType)
	}

	_, err := c.client.CreateUser(tracerCtx, username, userOptions.toArangoUserOptions())
	if err != nil {
		return err
	}

	return nil
}

// dropUser deletes a user from ArangoDB.
func (c *Client) dropUser(ctx context.Context, username string) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "dropUser", map[string]string{
		"user": username,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "dropUser", ID: username},
		time.Now(), "dropUser", span,
		observability.OperationLabels{Host: c.endpoint})

	err := c.client.RemoveUser(tracerCtx, username)
	if err != nil {
		return err
	}

	return err
}

// grantDB grants permissions for a database to a user.
func (c *Client) grantDB(ctx context.Context, database, username, permission string) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "grantDB", map[string]string{
		"DB": database,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "grantDB", Database: database, ID: username},
		time.Now(), "grantDB", span,
		observability.OperationLabels{Host: c.endpoint})

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetDatabaseAccess(tracerCtx, database, arangodb.Grant(permission))

	return err
}

// grantCollection grants permissions for a collection to a user.
func (c *Client) grantCollection(ctx context.Context, database, collection, username, permission string) error {
	tracerCtx, span := c.instrumentation.AddTrace(ctx, "GrantCollection", map[string]string{
		"collection": collection,
	})

	defer c.instrumentation.OperationStats(ctx,
		&QueryLog{Operation: "GrantCollection", Database: database, Collection: collection, ID: username},
		time.Now(), "GrantCollection", span,
		observability.OperationLabels{Host: c.endpoint, Table: collection})

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetCollectionAccess(tracerCtx, database, collection, arangodb.Grant(permission))

	return err
}
