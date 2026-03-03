package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func (c *Client) user(ctx context.Context, username string) (arangodb.User, error) {
	return c.client.User(ctx, username)
}

func (c *Client) database(ctx context.Context, name string) (arangodb.Database, error) {
	return c.client.GetDatabase(ctx, name, nil)
}

// createUser creates a new user in ArangoDB.
func (c *Client) createUser(ctx context.Context, username string, options any) error {
	ql := &QueryLog{Operation: "createUser", Host: c.endpoint, ID: username}

	tracerCtx, span := c.instrumentation.AddTrace(ctx, ql)

	defer c.instrumentation.OperationStats(ctx, ql, time.Now(), span)

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
	ql := &QueryLog{Operation: "dropUser", Host: c.endpoint, ID: username}

	tracerCtx, span := c.instrumentation.AddTrace(ctx, ql)

	defer c.instrumentation.OperationStats(ctx, ql, time.Now(), span)

	err := c.client.RemoveUser(tracerCtx, username)
	if err != nil {
		return err
	}

	return err
}

// grantDB grants permissions for a database to a user.
func (c *Client) grantDB(ctx context.Context, database, username, permission string) error {
	ql := &QueryLog{Operation: "grantDB", Host: c.endpoint, Database: database, ID: username}

	tracerCtx, span := c.instrumentation.AddTrace(ctx, ql)

	defer c.instrumentation.OperationStats(ctx, ql, time.Now(), span)

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetDatabaseAccess(tracerCtx, database, arangodb.Grant(permission))

	return err
}

// grantCollection grants permissions for a collection to a user.
func (c *Client) grantCollection(ctx context.Context, database, collection, username, permission string) error {
	ql := &QueryLog{Operation: "grantCollection", Host: c.endpoint, Database: database, Collection: collection, ID: username}

	tracerCtx, span := c.instrumentation.AddTrace(ctx, ql)

	defer c.instrumentation.OperationStats(ctx, ql, time.Now(), span)

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetCollectionAccess(tracerCtx, database, collection, arangodb.Grant(permission))

	return err
}
