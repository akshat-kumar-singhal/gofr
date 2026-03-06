package arangodb

import (
	"context"
	"errors"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gofr.dev/pkg/gofr/datasource/observability"

	"gofr.dev/pkg/gofr/datasource/arangodb/mocks"
)

var (
	errUserNotFound     = errors.New("user not found")
	errDBNotFound       = errors.New("database not found")
	errDocumentNotFound = errors.New("document not found")
)

// setupTestClient creates a test client with mock dependencies.
// It returns the client, mock ArangoDB client, and the controller for setting up expectations.
// The caller should defer ctrl.Finish() after calling this function.
func setupTestClient(t *testing.T, expectedOp string) (*Client, *mocks.MockClient, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockArango := mocks.NewMockClient(ctrl)
	mockInstr := setupMockInstrumenter(t, ctrl, expectedOp, 1)

	client := &Client{
		client:          mockArango,
		instrumentation: mockInstr,
		endpoint:        "http://localhost:8529",
	}

	return client, mockArango, ctrl
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
			assert.Equal(t, expectedOperation, q.GetOperation())
			return ctx, func() {}
		}).Times(count)

	// Allow Debugf calls (used for "already exists" scenarios)
	mockInstr.EXPECT().
		Debugf(gomock.Any(), gomock.Any()).
		AnyTimes()

	return mockInstr
}

// MockQueryCursor implements arangodb.Cursor for testing.
type MockQueryCursor struct {
	ctrl *gomock.Controller
	data []map[string]any
	idx  int
}

func NewMockQueryCursor(ctrl *gomock.Controller, data []map[string]any) *MockQueryCursor {
	return &MockQueryCursor{
		ctrl: ctrl,
		data: data,
		idx:  0,
	}
}

func (*MockQueryCursor) Close() error {
	return nil
}

func (*MockQueryCursor) CloseWithContext(_ context.Context) error {
	return nil
}

func (m *MockQueryCursor) HasMore() bool {
	return m.idx < len(m.data)
}

func (m *MockQueryCursor) ReadDocument(_ context.Context, document any) (arangodb.DocumentMeta, error) {
	if m.idx >= len(m.data) {
		return arangodb.DocumentMeta{}, shared.NoMoreDocumentsError{}
	}

	doc, ok := document.(*map[string]any)
	if !ok {
		return arangodb.DocumentMeta{}, errInvalidEdgeDocumentType
	}

	*doc = m.data[m.idx]
	meta := arangodb.DocumentMeta{}

	m.idx++

	return meta, nil
}

func (m *MockQueryCursor) Count() int64 {
	return int64(len(m.data))
}

func (*MockQueryCursor) Statistics() arangodb.CursorStats {
	return arangodb.CursorStats{}
}

func (*MockQueryCursor) Plan() arangodb.CursorPlan {
	return arangodb.CursorPlan{}
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name     string
		config   Config
		validate func(t *testing.T, client *Client)
	}{
		{
			name: "ValidConfig",
			config: Config{
				Host:     "localhost",
				Port:     8529,
				User:     "root",
				Password: "password",
			},
			validate: func(t *testing.T, client *Client) {
				t.Helper()
				require.NotNil(t, client)
				require.NotNil(t, client.config)
				require.Equal(t, "localhost", client.config.Host)
				require.Equal(t, 8529, client.config.Port)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := New(tc.config)
			tc.validate(t, client)
		})
	}
}

func TestClient_Connect(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
	}{
		{
			name: "ConnectWithValidConfig",
			config: Config{
				Host:     "localhost",
				Port:     8529,
				User:     "admin",
				Password: "root",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := New(tc.config)
			client.Connect()
			require.NotNil(t, client)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid config",
			config: Config{
				Host:     "localhost",
				Port:     8529,
				User:     "root",
				Password: "password",
			},
			expectErr: false,
		},
		{
			name: "Empty host",
			config: Config{
				Port:     8529,
				User:     "root",
				Password: "password",
			},
			expectErr: true,
			errMsg:    "missing required field in config: host is empty",
		},
		{
			name: "Empty port",
			config: Config{
				Host:     "localhost",
				User:     "root",
				Password: "password",
			},
			expectErr: true,
			errMsg:    "missing required field in config: port is empty",
		},
		{
			name: "Empty user",
			config: Config{
				Host:     "localhost",
				Port:     8529,
				Password: "password",
			},
			expectErr: true,
			errMsg:    "missing required field in config: user is empty",
		},
		{
			name: "Empty password",
			config: Config{
				Host: "localhost",
				Port: 8529,
				User: "root",
			},
			expectErr: true,
			errMsg:    "missing required field in config: password is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{config: &tc.config}
			err := client.validateConfig()

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:funlen // table-driven test with multiple test cases
func TestClient_Query(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		query          string
		bindVars       map[string]any
		queryOpts      []map[string]any
		setupMocks     func(test *TestGraph, ctrl *gomock.Controller)
		expectedResult []map[string]any
		expectedError  error
	}{
		{
			name:       "Success",
			expectedOp: "query",
			dbName:     "testDB",
			query:      "FOR doc IN collection RETURN doc",
			bindVars:   map[string]any{"key": "value"},
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				expectedResult := []map[string]any{
					{"_key": "doc1", "value": "test1"},
					{"_key": "doc2", "value": "test2"},
				}

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).
					Return(test.MockDB, nil)
				test.MockDB.EXPECT().Query(gomock.Any(), "FOR doc IN collection RETURN doc",
					&arangodb.QueryOptions{BindVars: map[string]any{"key": "value"}}).
					Return(NewMockQueryCursor(ctrl, expectedResult), nil)
			},
			expectedResult: []map[string]any{
				{"_key": "doc1", "value": "test1"},
				{"_key": "doc2", "value": "test2"},
			},
			expectedError: nil,
		},
		{
			name:       "WithBatchSizeAndFullCount",
			expectedOp: "query",
			dbName:     "testDB",
			query:      "FOR doc IN collection RETURN doc",
			bindVars:   map[string]any{"key": "value"},
			queryOpts: []map[string]any{{
				"batchSize": 50,
				"options": map[string]any{
					"fullCount": true,
				},
			}},
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				expectedResult := []map[string]any{
					{"_key": "doc1", "value": "v1"},
					{"_key": "doc2", "value": "v2"},
				}

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).
					Return(test.MockDB, nil)
				test.MockDB.EXPECT().
					Query(gomock.Any(), "FOR doc IN collection RETURN doc", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, opts *arangodb.QueryOptions) (arangodb.Cursor, error) {
						require.NotNil(t, opts)
						require.Equal(t, 50, opts.BatchSize)
						require.True(t, opts.Options.FullCount)

						return NewMockQueryCursor(ctrl, expectedResult), nil
					})
			},
			expectedResult: []map[string]any{
				{"_key": "doc1", "value": "v1"},
				{"_key": "doc2", "value": "v2"},
			},
			expectedError: nil,
		},
		{
			name:       "WithMaxPlans",
			expectedOp: "query",
			dbName:     "testDB",
			query:      "FOR doc IN collection RETURN doc",
			bindVars:   map[string]any{"key": "value"},
			queryOpts: []map[string]any{{
				"options": map[string]any{
					"maxPlans": 5,
				},
			}},
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				expectedResult := []map[string]any{
					{"_key": "doc1", "value": "v1"},
				}

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).
					Return(test.MockDB, nil)
				test.MockDB.EXPECT().
					Query(gomock.Any(), "FOR doc IN collection RETURN doc", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, opts *arangodb.QueryOptions) (arangodb.Cursor, error) {
						require.NotNil(t, opts)
						require.Equal(t, 5, opts.Options.MaxPlans)

						return NewMockQueryCursor(ctrl, expectedResult), nil
					})
			},
			expectedResult: []map[string]any{
				{"_key": "doc1", "value": "v1"},
			},
			expectedError: nil,
		},
		{
			name:       "InvalidResultType",
			expectedOp: "query",
			dbName:     "testDB",
			query:      "FOR doc IN collection RETURN doc",
			bindVars:   map[string]any{"key": "value"},
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).
					Return(test.MockDB, nil)
				test.MockDB.EXPECT().Query(gomock.Any(), "FOR doc IN collection RETURN doc", gomock.Any()).
					Return(NewMockQueryCursor(ctrl, nil), nil)
			},
			expectedError: errInvalidResultType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockDB := mocks.NewMockDatabase(ctrl)
			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOp, 1)

			client := &Client{
				client:          mockArango,
				instrumentation: mockInstr,
				endpoint:        "http://localhost:8529",
			}
			client.Document = &Document{client: client}

			test := &TestGraph{
				Ctrl:       ctrl,
				MockArango: mockArango,
				MockDB:     mockDB,
				Client:     client,
				Ctx:        context.Background(),
			}

			tc.setupMocks(test, ctrl)

			// Handle InvalidResultType case separately
			if tc.name == "InvalidResultType" {
				var result int

				err := client.Query(test.Ctx, tc.dbName, tc.query, tc.bindVars, &result)
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)

				return
			}

			var (
				result []map[string]any
				err    error
			)

			if len(tc.queryOpts) > 0 {
				err = client.Query(test.Ctx, tc.dbName, tc.query, tc.bindVars, &result, tc.queryOpts...)
			} else {
				err = client.Query(test.Ctx, tc.dbName, tc.query, tc.bindVars, &result)
			}

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestClient_HealthCheck(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(test *TestGraph)
		expectedStatus string
		expectedError  error
	}{
		{
			name: "Success",
			setupMocks: func(test *TestGraph) {
				expectedVersion := arangodb.VersionInfo{
					Version: "3.9.0",
					Server:  "arango",
				}
				test.MockArango.EXPECT().Version(test.Ctx).Return(expectedVersion, nil)
			},
			expectedStatus: "UP",
			expectedError:  nil,
		},
		{
			name: "Error",
			setupMocks: func(test *TestGraph) {
				test.MockArango.EXPECT().Version(test.Ctx).Return(arangodb.VersionInfo{}, errStatusDown)
			},
			expectedStatus: "DOWN",
			expectedError:  errStatusDown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test := setupGraphTest(t)
			defer test.Ctrl.Finish()

			tc.setupMocks(test)

			health, err := test.Client.HealthCheck(test.Ctx)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			h, ok := health.(*Health)
			require.True(t, ok)
			require.Equal(t, tc.expectedStatus, h.Status)
			require.Equal(t, test.Client.endpoint, h.Details["endpoint"])

			if tc.expectedStatus == "UP" {
				require.Equal(t, arangodb.Version("3.9.0"), h.Details["version"])
				require.Equal(t, "arango", h.Details["server"])
			}
		})
	}
}
