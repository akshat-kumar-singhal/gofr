package arangodb

import (
	"context"
	"fmt"

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
	ql := &QueryLog{Operation: "createUser", ID: username}

	tracerCtx, done := c.instrumentQuery(ctx, ql)
	defer done()

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
	ql := &QueryLog{Operation: "dropUser", ID: username}

	tracerCtx, done := c.instrumentQuery(ctx, ql)
	defer done()

	err := c.client.RemoveUser(tracerCtx, username)
	if err != nil {
		return err
	}

	return err
}

// grantDB grants permissions for a database to a user.
func (c *Client) grantDB(ctx context.Context, database, username, permission string) error {
	ql := &QueryLog{Operation: "grantDB", Database: database, ID: username}

	tracerCtx, done := c.instrumentQuery(ctx, ql)
	defer done()

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetDatabaseAccess(tracerCtx, database, arangodb.Grant(permission))

	return err
}

// grantCollection grants permissions for a collection to a user.
func (c *Client) grantCollection(ctx context.Context, database, collection, username, permission string) error {
	ql := &QueryLog{Operation: "grantCollection", Database: database, Collection: collection, ID: username}

	tracerCtx, done := c.instrumentQuery(ctx, ql)
	defer done()

	user, err := c.client.User(tracerCtx, username)
	if err != nil {
		return err
	}

	err = user.SetCollectionAccess(tracerCtx, database, collection, arangodb.Grant(permission))

	return err
}
