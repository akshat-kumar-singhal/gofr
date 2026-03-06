package mongo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.uber.org/mock/gomock"
	"gofr.dev/pkg/gofr/datasource/observability"
)

// duplicateKeyError returns a mock write error response for duplicate key errors.
func duplicateKeyError() bson.D {
	return mtest.CreateWriteErrorsResponse(mtest.WriteError{
		Index:   1,
		Code:    11000,
		Message: "duplicate key error",
	})
}

// newMockClient creates a Client with the given mtest.T database and instrumenter.
func newMockClient(mt *mtest.T, instr observability.Instrumenter) *Client {
	return &Client{
		Database:        mt.DB,
		instrumentation: instr,
		config:          &Config{Host: "localhost", Database: "test"},
	}
}

// setupMockInstrumenter creates a mock instrumenter with InstrumentOperation expectations.
// It accepts a gomock.Controller to use the same controller as the test.
// count controls the expected number of instrumentation calls:
// - count = 0: No instrumentation calls expected (validation fails before instrumentation)
// - count = 1: Expects one InstrumentOperation call.
func setupMockInstrumenter(t *testing.T, ctrl *gomock.Controller, expectedOperation string, count int) *observability.MockInstrumenter {
	t.Helper()

	mockInstr := observability.NewMockInstrumenter(ctrl)

	mockInstr.EXPECT().
		InstrumentOperation(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, q observability.ObservableQuery) (context.Context, func()) {
			if expectedOperation != "" {
				assert.Equal(t, expectedOperation, q.GetOperation())
			}

			return ctx, func() {}
		}).Times(count)

	return mockInstr
}

func Test_NewMongoClient(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		setupClient    func(*Client)
		expectDatabase bool
	}{
		{
			name:   "success",
			config: Config{Database: "test", Host: "localhost", Port: 27017, User: "admin", ConnectionTimeout: 1 * time.Second},
			setupClient: func(c *Client) {
				c.Database = &mongo.Database{}
			},
			expectDatabase: true,
		},
		{
			name:           "error_invalid_config",
			config:         Config{Host: "mongo", Database: "test"},
			setupClient:    nil,
			expectDatabase: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := New(tc.config)
			if tc.setupClient != nil {
				tc.setupClient(client)
			}

			client.Connect()

			if tc.expectDatabase {
				assert.NotNil(t, client)
			} else {
				assert.Nil(t, client.Database)
			}
		})
	}
}

func TestGenerateMongoURI(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedURI   string
		expectedHost  string
		expectedError string
	}{
		{
			name: "Valid Config",
			config: Config{
				User:     "admin",
				Password: "p@##word:",
				Host:     "localhost",
				Port:     27017,
				Database: "mydb",
			},
			expectedURI:   "mongodb://admin:p%2540%2523%2523word%253A@localhost:27017/mydb?authSource=admin",
			expectedHost:  "localhost",
			expectedError: "",
		},
		{
			name: "Valid Config without authentication",
			config: Config{
				Host:     "localhost",
				Port:     27017,
				Database: "mydb",
			},
			expectedURI:   "mongodb://localhost:27017/mydb?authSource=admin",
			expectedHost:  "localhost",
			expectedError: "",
		},
		{
			name: "Predefined URI",
			config: Config{
				URI: "mongodb://admin:password@localhost:27017/mydb?authSource=admin",
			},
			expectedURI:   "mongodb://admin:password@localhost:27017/mydb?authSource=admin",
			expectedHost:  "localhost",
			expectedError: "",
		},
		{
			name: "Empty Host",
			config: Config{
				User:     "admin",
				Password: "password",
				Port:     27017,
				Database: "mydb",
			},
			expectedURI:   "",
			expectedHost:  "",
			expectedError: "missing required field in config: host is empty",
		},
		{
			name: "Invalid Port",
			config: Config{
				User:     "admin",
				Password: "password",
				Host:     "localhost",
				Database: "mydb",
			},
			expectedURI:   "",
			expectedHost:  "",
			expectedError: "missing required field in config: port is empty",
		},
		{
			name: "Empty Database",
			config: Config{
				User:     "admin",
				Password: "password",
				Host:     "localhost",
				Port:     27017,
			},
			expectedURI:   "",
			expectedHost:  "",
			expectedError: "missing required field in config: database is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := Client{config: &test.config}
			uri, host, err := generateMongoURI(client.config)

			assert.Equal(t, test.expectedURI, uri, "Unexpected URI")
			assert.Equal(t, test.expectedHost, host, "Unexpected Host")

			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError, "Unexpected error message")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
		})
	}
}

func TestGetDBHost(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expected    string
		expectedErr string
	}{
		{
			name:        "Valid URI with host and port",
			uri:         "mongodb://username:password@hostname:27017/database?authSource=admin",
			expected:    "hostname",
			expectedErr: "",
		},
		{
			name:        "Valid URI with IP address as host",
			uri:         "mongodb://username:password@192.168.1.1:27017/database?authSource=admin",
			expected:    "192.168.1.1",
			expectedErr: "",
		},
		{
			name:        "Invalid URI with no host",
			uri:         "mongodb://username:password@:27017/database?authSource=admin",
			expected:    "",
			expectedErr: "failed to parse host from MongoDB URI",
		},
		{
			name:        "Empty URI",
			uri:         "",
			expected:    "",
			expectedErr: "parse \"\": empty url",
		},
		{
			name:        "Malformed URI",
			uri:         "mongodb:/username:password@hostname:27017/database?authSource=admin",
			expected:    "",
			expectedErr: "failed to parse host from MongoDB URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := getDBHost(tt.uri)

			assert.Equal(t, tt.expected, host)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}

func Test_InsertOne(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	tests := []struct {
		name              string
		mockResponse      func(mt *mtest.T)
		expectError       bool
		expectNilRes      bool
		expectedOperation string
	}{
		{
			name:              "success",
			mockResponse:      func(mt *mtest.T) { mt.AddMockResponses(mtest.CreateSuccessResponse()) },
			expectError:       false,
			expectNilRes:      false,
			expectedOperation: "insertOne",
		},
		{
			name:         "error",
			mockResponse: func(mt *mtest.T) { mt.AddMockResponses(duplicateKeyError()) },
			expectError:  true,
			expectNilRes: true,
		},
	}

	for _, tc := range tests {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOperation, 1)
			cl := newMockClient(mt, mockInstr)

			tc.mockResponse(mt)

			doc := map[string]any{"name": "Aryan"}
			resp, err := cl.InsertOne(context.Background(), mt.Coll.Name(), doc)

			if tc.expectNilRes {
				assert.Nil(t, resp)
			} else {
				assert.NotNil(t, resp)
			}

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_InsertMany(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	tests := []struct {
		name         string
		mockResponse func(mt *mtest.T)
		expectError  bool
		expectNilRes bool
	}{
		{
			name:         "success",
			mockResponse: func(mt *mtest.T) { mt.AddMockResponses(mtest.CreateSuccessResponse()) },
			expectError:  false,
			expectNilRes: false,
		},
		{
			name:         "error",
			mockResponse: func(mt *mtest.T) { mt.AddMockResponses(duplicateKeyError()) },
			expectError:  true,
			expectNilRes: true,
		},
	}

	for _, tc := range tests {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInstr := setupMockInstrumenter(t, ctrl, "", 1)
			cl := newMockClient(mt, mockInstr)

			tc.mockResponse(mt)

			doc := map[string]any{"name": "Aryan"}
			resp, err := cl.InsertMany(context.Background(), mt.Coll.Name(), []any{doc, doc})

			if tc.expectNilRes {
				assert.Nil(t, resp)
			} else {
				assert.NotNil(t, resp)
			}

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CreateCollection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("createCollection", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "createCollection", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := cl.CreateCollection(context.Background(), mt.Coll.Name())

		require.NoError(t, err)
	})
}

func Test_FindMultipleCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("FindSuccess", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "find", 1)
		cl := newMockClient(mt, mockInstr)

		var foundDocuments []any

		id1 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: id1},
			{Key: "name", Value: "john"},
			{Key: "email", Value: "john.doe@test.com"},
		})

		killCursors := mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch)
		mt.AddMockResponses(first, killCursors)

		mt.AddMockResponses(first)

		err := cl.Find(context.Background(), mt.Coll.Name(), bson.D{{}}, &foundDocuments)

		assert.NoError(t, err, "Unexpected error during Find operation")
	})

	mt.Run("FindCursorError", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "find", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := cl.Find(context.Background(), mt.Coll.Name(), bson.D{{}}, nil)

		require.ErrorContains(t, err, "database response does not contain a cursor")
	})

	mt.Run("FindCursorParseError", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "find", 1)
		cl := newMockClient(mt, mockInstr)

		var foundDocuments []any

		id1 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: id1},
			{Key: "name", Value: "john"},
			{Key: "email", Value: "john.doe@test.com"},
		})

		mt.AddMockResponses(first)

		mt.AddMockResponses(first)

		err := cl.Find(context.Background(), mt.Coll.Name(), bson.D{{}}, &foundDocuments)

		require.ErrorContains(t, err, "cursor.nextBatch should be an array but is a BSON invalid")
	})
}

func Test_FindOneCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("FindOneSuccess", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "findOne", 1)
		cl := newMockClient(mt, mockInstr)

		type user struct {
			ID    primitive.ObjectID
			Name  string
			Email string
		}

		var foundDocuments user

		expectedUser := user{
			ID:    primitive.NewObjectID(),
			Name:  "john",
			Email: "john.doe@test.com",
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: expectedUser.ID},
			{Key: "name", Value: expectedUser.Name},
			{Key: "email", Value: expectedUser.Email},
		}))

		err := cl.FindOne(context.Background(), mt.Coll.Name(), bson.D{{}}, &foundDocuments)

		assert.Equal(t, expectedUser.Name, foundDocuments.Name)
		assert.NoError(t, err)
	})

	mt.Run("FindOneError", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "findOne", 1)
		cl := newMockClient(mt, mockInstr)

		type user struct {
			ID    primitive.ObjectID
			Name  string
			Email string
		}

		var foundDocuments user

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch))

		err := cl.FindOne(context.Background(), mt.Coll.Name(), bson.D{{}}, &foundDocuments)

		assert.Error(t, err)
	})
}

func Test_UpdateByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "updateByID", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		resp, err := cl.UpdateByID(context.Background(), mt.Coll.Name(), "1", bson.M{"$set": bson.M{"name": "test"}})

		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})
}

func Test_UpdateOne(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "updateOne", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := cl.UpdateOne(context.Background(), mt.Coll.Name(), bson.D{{Key: "name", Value: "test"}}, bson.M{"$set": bson.M{"name": "testing"}})

		assert.NoError(t, err)
	})
}

func Test_UpdateMany(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "updateMany", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		_, err := cl.UpdateMany(context.Background(), mt.Coll.Name(), bson.D{{Key: "name", Value: "test"}},
			bson.M{"$set": bson.M{"name": "testing"}})

		assert.NoError(t, err)
	})
}

func Test_CountDocuments(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("countDocuments", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "countDocuments", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.restaurants", mtest.FirstBatch, bson.D{{Key: "n", Value: 1}}))

		// For count to work, mongo needs an index. So we need to create that. Index view should contain a key. Value does not matter
		indexView := mt.Coll.Indexes()
		_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
			Keys: bson.D{{Key: "x", Value: 1}},
		})

		require.NoError(mt, err, "CreateOne error for index: %v", err)

		resp, err := cl.CountDocuments(context.Background(), mt.Coll.Name(), bson.D{{Key: "name", Value: "test"}})

		assert.Equal(t, int64(1), resp)
		assert.NoError(t, err)
	})
}

func Test_DeleteOne(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	tests := []struct {
		name              string
		mockResponse      func(mt *mtest.T)
		expectError       bool
		expectedCount     int64
		expectedOperation string
	}{
		{
			name:              "success",
			mockResponse:      func(mt *mtest.T) { mt.AddMockResponses(mtest.CreateSuccessResponse()) },
			expectError:       false,
			expectedCount:     0,
			expectedOperation: "deleteOne",
		},
		{
			name:          "error",
			mockResponse:  func(mt *mtest.T) { mt.AddMockResponses(duplicateKeyError()) },
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOperation, 1)
			cl := newMockClient(mt, mockInstr)

			tc.mockResponse(mt)

			resp, err := cl.DeleteOne(context.Background(), mt.Coll.Name(), bson.D{{}})

			assert.Equal(t, tc.expectedCount, resp)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_DeleteMany(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	tests := []struct {
		name          string
		mockResponse  func(mt *mtest.T)
		expectError   bool
		expectedCount int64
	}{
		{
			name:          "success",
			mockResponse:  func(mt *mtest.T) { mt.AddMockResponses(mtest.CreateSuccessResponse()) },
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:          "error",
			mockResponse:  func(mt *mtest.T) { mt.AddMockResponses(duplicateKeyError()) },
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInstr := setupMockInstrumenter(t, ctrl, "", 1)
			cl := newMockClient(mt, mockInstr)

			tc.mockResponse(mt)

			resp, err := cl.DeleteMany(context.Background(), mt.Coll.Name(), bson.D{{}})

			assert.Equal(t, tc.expectedCount, resp)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Drop(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("Drop", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInstr := setupMockInstrumenter(t, ctrl, "drop", 1)
		cl := newMockClient(mt, mockInstr)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := cl.Drop(context.Background(), mt.Coll.Name())

		assert.NoError(t, err)
	})
}

func TestClient_StartSession(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("StartSessionCommitTransactionSuccess", func(mt *mtest.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// StartSession + InsertOne = 2 instrumentation calls
		mockInstr := setupMockInstrumenter(t, ctrl, "", 2)
		cl := newMockClient(mt, mockInstr)

		// Add mock responses if necessary
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		// Call the StartSession method
		sess, err := cl.StartSession()

		ses, ok := sess.(Transaction)
		if ok {
			err = ses.StartTransaction()
		}

		require.NoError(t, err)

		cl.Database = mt.DB
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		doc := map[string]any{"name": "Aryan"}

		resp, err := cl.InsertOne(context.Background(), mt.Coll.Name(), doc)

		assert.NotNil(t, resp)
		require.NoError(t, err)

		err = ses.CommitTransaction(context.Background())

		require.NoError(t, err)

		ses.EndSession(context.Background())

		// Assert that there was no error
		require.NoError(t, err)
	})
}

func Test_HealthCheck(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	tests := []struct {
		name           string
		mockResponse   func(mt *mtest.T)
		expectedStatus string
		expectedErr    error
	}{
		{
			name:           "success",
			mockResponse:   func(mt *mtest.T) { mt.AddMockResponses(mtest.CreateSuccessResponse()) },
			expectedStatus: "UP",
			expectedErr:    nil,
		},
		{
			name:           "error",
			mockResponse:   func(mt *mtest.T) { mt.AddMockResponses(duplicateKeyError()) },
			expectedStatus: "DOWN",
			expectedErr:    errStatusDown,
		},
	}

	for _, tc := range tests {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInstr := setupMockInstrumenter(t, ctrl, "", 0)
			cl := newMockClient(mt, mockInstr)

			tc.mockResponse(mt)

			resp, err := cl.HealthCheck(context.Background())

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}

			assert.Contains(t, fmt.Sprint(resp), tc.expectedStatus)
		})
	}
}
