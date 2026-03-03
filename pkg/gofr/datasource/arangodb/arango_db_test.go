package arangodb

import (
	"context"
	"errors"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gofr.dev/pkg/gofr/datasource/arangodb/mocks"
)

var errCollectionNotFound = errors.New("collection not found")

func TestClient_CreateDB(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		dbName        string
		setupMocks    func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "createDB",
			dbName:     "testDB",
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().DatabaseExists(gomock.Any(), "testDB").Return(false, nil)
				mockArango.EXPECT().CreateDatabase(gomock.Any(), "testDB", nil).Return(nil, nil)
			},
			expectedError: nil,
		},
		{
			name:       "Error_CreateFails",
			expectedOp: "createDB",
			dbName:     "errorDB",
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().DatabaseExists(gomock.Any(), "errorDB").Return(false, nil)
				mockArango.EXPECT().CreateDatabase(gomock.Any(), "errorDB", nil).Return(nil, errDBNotFound)
			},
			expectedError: errDBNotFound,
		},
		{
			name:       "Error_AlreadyExists",
			expectedOp: "createDB",
			dbName:     "dbExists",
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().DatabaseExists(gomock.Any(), "dbExists").Return(true, nil)
			},
			expectedError: ErrDatabaseExists,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOp, 1)

			client := &Client{
				client:          mockArango,
				instrumentation: mockInstr,
				endpoint:        "http://localhost:8529",
			}
			client.DB = &DB{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.CreateDB(context.Background(), tc.dbName)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_DropDB(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		dbName        string
		setupMocks    func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "dropDB",
			dbName:     "testDB",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", &arangodb.GetDatabaseOptions{}).
					Return(arangodb.Database(mockDB), nil)
				mockDB.EXPECT().Remove(gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Error_DBNotFound",
			expectedOp: "dropDB",
			dbName:     "testDB",
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", &arangodb.GetDatabaseOptions{}).
					Return(nil, errDBNotFound)
			},
			expectedError: errDBNotFound,
		},
		{
			name:       "Error_RemoveFails",
			expectedOp: "dropDB",
			dbName:     "testDB",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", &arangodb.GetDatabaseOptions{}).
					Return(mockDB, nil)
				mockDB.EXPECT().Remove(gomock.Any()).Return(errDBNotFound)
			},
			expectedError: errDBNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOp, 1)

			client := &Client{
				client:          mockArango,
				instrumentation: mockInstr,
				endpoint:        "http://localhost:8529",
			}
			client.DB = &DB{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.DropDB(context.Background(), tc.dbName)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_CreateCollection(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		isEdge         bool
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "createCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			isEdge:         true,
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().CollectionExists(gomock.Any(), "testCollection").Return(false, nil)
				mockDB.EXPECT().CreateCollectionV2(gomock.Any(), "testCollection", gomock.Any()).Return(nil, nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_CreateFails",
			expectedOp:     "createCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			isEdge:         false,
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().CollectionExists(gomock.Any(), "testCollection").Return(false, nil)
				mockDB.EXPECT().CreateCollectionV2(gomock.Any(), "testCollection", gomock.Any()).Return(nil, errCollectionNotFound)
			},
			expectedError: errCollectionNotFound,
		},
		{
			name:           "Error_AlreadyExists",
			expectedOp:     "createCollection",
			dbName:         "dbExists",
			collectionName: "testCollection",
			isEdge:         true,
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "dbExists", nil).Return(mockDB, nil)
				mockDB.EXPECT().CollectionExists(gomock.Any(), "testCollection").Return(true, nil)
			},
			expectedError: ErrCollectionExists,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOp, 1)

			client := &Client{
				client:          mockArango,
				instrumentation: mockInstr,
				endpoint:        "http://localhost:8529",
			}
			client.DB = &DB{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.CreateCollection(context.Background(), tc.dbName, tc.collectionName, tc.isEdge)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_DropCollection(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "dropCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil)
				mockCollection.EXPECT().Remove(gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_CollectionNotFound",
			expectedOp:     "dropCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(nil, errCollectionNotFound)
			},
			expectedError: errCollectionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockInstr := setupMockInstrumenter(t, ctrl, tc.expectedOp, 1)

			client := &Client{
				client:          mockArango,
				instrumentation: mockInstr,
				endpoint:        "http://localhost:8529",
			}
			client.DB = &DB{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.DropCollection(context.Background(), tc.dbName, tc.collectionName)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
