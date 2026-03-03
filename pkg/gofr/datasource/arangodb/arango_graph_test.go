package arangodb

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gofr.dev/pkg/gofr/datasource/arangodb/mocks"
)

// TestGraph represents the test environment for graph-related tests.
type TestGraph struct {
	Ctrl       *gomock.Controller
	MockArango *mocks.MockClient
	MockDB     *mocks.MockDatabase
	Client     *Client
	Graph      *Graph
	Ctx        context.Context
	DBName     string
	GraphName  string
	EdgeDefs   *EdgeDefinition
}

// setupGraphTest creates a new test environment for graph tests.
func setupGraphTest(t *testing.T) *TestGraph {
	t.Helper()

	return setupGraphTestWithInstr(t, "", 0)
}

// setupGraphTestWithInstr creates a new test environment for graph tests with custom instrumentation settings.
func setupGraphTestWithInstr(t *testing.T, expectedOp string, count int) *TestGraph {
	t.Helper()
	ctrl := gomock.NewController(t)

	mockArango := mocks.NewMockClient(ctrl)
	mockDB := mocks.NewMockDatabase(ctrl)
	mockInstr := setupMockInstrumenter(t, ctrl, expectedOp, count)

	client := &Client{
		instrumentation: mockInstr,
		client:          mockArango,
		endpoint:        "http://localhost:8529",
	}

	client.Graph = &Graph{client: client}
	ctx := context.Background()

	return &TestGraph{
		Ctrl:       ctrl,
		MockArango: mockArango,
		MockDB:     mockDB,
		Client:     client,
		Graph:      client.Graph,
		Ctx:        ctx,
		DBName:     "testDB",
		GraphName:  "testGraph",
		EdgeDefs:   &EdgeDefinition{{Collection: "edgeColl", From: []string{"fromColl"}, To: []string{"toColl"}}},
	}
}

func TestGraph_CreateGraph(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		dbName        string
		graphName     string
		setupMocks    func(test *TestGraph, ctrl *gomock.Controller)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "createGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				mockGraph := mocks.NewMockGraph(ctrl)
				graphInterface := arangodb.Graph(mockGraph)

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().GraphExists(gomock.Any(), "testGraph").Return(false, nil)
				test.MockDB.EXPECT().CreateGraph(gomock.Any(), "testGraph", gomock.Any(), nil).Return(graphInterface, nil)
			},
			expectedError: nil,
		},
		{
			name:       "Error_AlreadyExists",
			expectedOp: "createGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, _ *gomock.Controller) {
				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().GraphExists(gomock.Any(), "testGraph").Return(true, nil)
			},
			expectedError: ErrGraphExists,
		},
		{
			name:       "Error_CreateFails",
			expectedOp: "createGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, _ *gomock.Controller) {
				options := &arangodb.GraphDefinition{EdgeDefinitions: []arangodb.EdgeDefinition{{
					Collection: "edgeColl",
					From:       []string{"fromColl"},
					To:         []string{"toColl"},
				}}}

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().GraphExists(gomock.Any(), "testGraph").Return(false, nil)
				test.MockDB.EXPECT().CreateGraph(gomock.Any(), "testGraph", options, nil).Return(nil, errInvalidEdgeDocumentType)
			},
			expectedError: errInvalidEdgeDocumentType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test := setupGraphTestWithInstr(t, tc.expectedOp, 1)
			defer test.Ctrl.Finish()

			tc.setupMocks(test, test.Ctrl)

			err := test.Client.CreateGraph(test.Ctx, tc.dbName, tc.graphName, test.EdgeDefs)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGraph_DropGraph(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		dbName        string
		graphName     string
		setupMocks    func(test *TestGraph, ctrl *gomock.Controller)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "dropGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				mockGraph := mocks.NewMockGraph(ctrl)
				graphInterface := arangodb.Graph(mockGraph)

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().Graph(gomock.Any(), "testGraph", nil).Return(graphInterface, nil)
				mockGraph.EXPECT().Remove(gomock.Any(), &arangodb.RemoveGraphOptions{DropCollections: true}).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Error_DBNotFound",
			expectedOp: "dropGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, _ *gomock.Controller) {
				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(nil, errDBNotFound)
			},
			expectedError: errDBNotFound,
		},
		{
			name:       "Error_RemoveFails",
			expectedOp: "dropGraph",
			dbName:     "testDB",
			graphName:  "testGraph",
			setupMocks: func(test *TestGraph, ctrl *gomock.Controller) {
				mockGraph := mocks.NewMockGraph(ctrl)
				graphInterface := arangodb.Graph(mockGraph)

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().Graph(gomock.Any(), "testGraph", nil).Return(graphInterface, nil)
				mockGraph.EXPECT().Remove(gomock.Any(), &arangodb.RemoveGraphOptions{DropCollections: true}).Return(errStatusDown)
			},
			expectedError: errStatusDown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test := setupGraphTestWithInstr(t, tc.expectedOp, 1)
			defer test.Ctrl.Finish()

			tc.setupMocks(test, test.Ctrl)

			err := test.Client.DropGraph(test.Ctx, tc.dbName, tc.graphName)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_GetEdges(t *testing.T) {
	testCases := []struct {
		name           string
		dbName         string
		graphName      string
		edgeCollection string
		vertexID       string
		setupMocks     func(test *TestGraph, ctrl *gomock.Controller)
		skipInstr      bool // true for tests where validation fails before instrumentation
		expectedEdges  []arangodb.EdgeDetails
		expectedError  error
	}{
		{
			name:           "Success",
			dbName:         "testDB",
			graphName:      "testGraph",
			edgeCollection: "edgeColl",
			vertexID:       "vertexID",
			setupMocks: func(test *TestGraph, _ *gomock.Controller) {
				expectedEdges := []arangodb.EdgeDetails{{
					To:    "toColl",
					From:  "fromColl",
					Label: "label",
				}}

				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(test.MockDB, nil)
				test.MockDB.EXPECT().GetEdges(gomock.Any(), "edgeColl", "vertexID", nil).Return(expectedEdges, nil)
			},
			expectedEdges: []arangodb.EdgeDetails{{
				To:    "toColl",
				From:  "fromColl",
				Label: "label",
			}},
			expectedError: nil,
		},
		{
			name:           "Error_DBNotFound",
			dbName:         "testDB",
			graphName:      "testGraph",
			edgeCollection: "edgeColl",
			vertexID:       "vertexID",
			setupMocks: func(test *TestGraph, _ *gomock.Controller) {
				test.MockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(nil, errDBNotFound)
			},
			expectedError: errDBNotFound,
		},
		{
			name:           "Error_InvalidInput",
			dbName:         "testDB",
			graphName:      "testGraph",
			edgeCollection: "",
			vertexID:       "",
			setupMocks:     func(_ *TestGraph, _ *gomock.Controller) {},
			skipInstr:      true, // Validation fails before instrumentation
			expectedError:  errInvalidInput,
		},
		{
			name:           "Error_InvalidResponseType",
			dbName:         "testDB",
			graphName:      "testGraph",
			edgeCollection: "edgeColl",
			vertexID:       "vertexID",
			setupMocks:     func(_ *TestGraph, _ *gomock.Controller) {},
			skipInstr:      true, // Validation fails before instrumentation
			expectedError:  errInvalidResponseType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count := 1
			if tc.skipInstr {
				count = 0
			}

			test := setupGraphTestWithInstr(t, "getEdges", count)
			defer test.Ctrl.Finish()

			tc.setupMocks(test, test.Ctrl)

			// Special handling for InvalidResponseType test
			if tc.name == "Error_InvalidResponseType" {
				var resp string

				err := test.Client.GetEdges(test.Ctx, tc.dbName, tc.graphName, tc.edgeCollection, tc.vertexID, &resp)
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)

				return
			}

			var resp EdgeDetails

			err := test.Client.GetEdges(test.Ctx, tc.dbName, tc.graphName, tc.edgeCollection, tc.vertexID, &resp)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedEdges, []arangodb.EdgeDetails(resp))
			}
		})
	}
}
