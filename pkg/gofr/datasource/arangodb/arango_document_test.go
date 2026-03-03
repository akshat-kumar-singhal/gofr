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

func TestClient_CreateDocument(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		document       any
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedKey    string
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "createDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			document:       "testDocument",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().Properties(gomock.Any()).Return(arangodb.CollectionProperties{}, nil)
				mockCollection.EXPECT().CreateDocument(gomock.Any(), "testDocument").
					Return(arangodb.CollectionDocumentCreateResponse{DocumentMeta: arangodb.DocumentMeta{Key: "testDocument", ID: "1"}}, nil)
			},
			expectedKey:   "testDocument",
			expectedError: nil,
		},
		{
			name:           "Error_CreateFails",
			expectedOp:     "createDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			document:       "testDocument",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().Properties(gomock.Any()).Return(arangodb.CollectionProperties{}, nil)
				mockCollection.EXPECT().CreateDocument(gomock.Any(), "testDocument").
					Return(arangodb.CollectionDocumentCreateResponse{}, errDocumentNotFound)
			},
			expectedKey:   "",
			expectedError: errDocumentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			client.DB = &DB{client: client}
			client.Document = &Document{client: client}

			tc.setupMocks(ctrl, mockArango)

			docKey, err := client.CreateDocument(context.Background(), tc.dbName, tc.collectionName, tc.document)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				require.Empty(t, docKey)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedKey, docKey)
			}
		})
	}
}

func TestClient_GetDocument(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		documentID     string
		result         any
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "getDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			result:         "",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().ReadDocument(gomock.Any(), "testDocument", "").
					Return(arangodb.DocumentMeta{Key: "testKey", ID: "1"}, nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_DocumentNotFound",
			expectedOp:     "getDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			result:         "",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().ReadDocument(gomock.Any(), "testDocument", "").
					Return(arangodb.DocumentMeta{}, errDocumentNotFound)
			},
			expectedError: errDocumentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			client.DB = &DB{client: client}
			client.Document = &Document{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.GetDocument(context.Background(), tc.dbName, tc.collectionName, tc.documentID, tc.result)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_UpdateDocument(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		documentID     string
		document       map[string]any
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "updateDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			document:       map[string]any{"field": "value"},
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)
				document := map[string]any{"field": "value"}

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().UpdateDocument(gomock.Any(), "testDocument", document).
					Return(arangodb.CollectionDocumentUpdateResponse{
						DocumentMetaWithOldRev: arangodb.DocumentMetaWithOldRev{DocumentMeta: arangodb.DocumentMeta{Key: "testKey", ID: "1", Rev: ""}}}, nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_UpdateFails",
			expectedOp:     "updateDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			document:       map[string]any{"field": "value"},
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)
				document := map[string]any{"field": "value"}

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().UpdateDocument(gomock.Any(), "testDocument", document).
					Return(arangodb.CollectionDocumentUpdateResponse{}, errDocumentNotFound)
			},
			expectedError: errDocumentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			client.DB = &DB{client: client}
			client.Document = &Document{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.UpdateDocument(context.Background(), tc.dbName, tc.collectionName, tc.documentID, tc.document)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_DeleteDocument(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		documentID     string
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "deleteDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().DeleteDocument(gomock.Any(), "testDocument").
					Return(arangodb.CollectionDocumentDeleteResponse{DocumentMeta: arangodb.DocumentMeta{Key: "testKey", ID: "1", Rev: ""}}, nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_DeleteFails",
			expectedOp:     "deleteDocument",
			dbName:         "testDB",
			collectionName: "testCollection",
			documentID:     "testDocument",
			setupMocks: func(ctrl *gomock.Controller, mockArango *mocks.MockClient) {
				mockDB := mocks.NewMockDatabase(ctrl)
				mockCollection := mocks.NewMockCollection(ctrl)

				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil).AnyTimes()
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(mockCollection, nil).AnyTimes()
				mockCollection.EXPECT().DeleteDocument(gomock.Any(), "testDocument").
					Return(arangodb.CollectionDocumentDeleteResponse{}, errDocumentNotFound)
			},
			expectedError: errDocumentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			client.DB = &DB{client: client}
			client.Document = &Document{client: client}

			tc.setupMocks(ctrl, mockArango)

			err := client.DeleteDocument(context.Background(), tc.dbName, tc.collectionName, tc.documentID)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecuteCollectionOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArango := mocks.NewMockClient(ctrl)
	mockDatabase := mocks.NewMockDatabase(ctrl)
	mockCollection := mocks.NewMockCollection(ctrl)

	client := New(Config{Host: "localhost", Port: 8527, User: "root", Password: "root"})

	client.client = mockArango
	d := Document{client: client}

	ctx := context.Background()
	dbName := "testDB"
	collectionName := "testCollection"
	operation := "createDocument"
	documentID := "doc123"

	mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).
		Return(mockDatabase, nil).AnyTimes()
	mockDatabase.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).
		Return(mockCollection, nil).AnyTimes()

	_, _, err := executeCollectionOperation(ctx, d, dbName, collectionName, operation, documentID)
	require.NoError(t, err)
}

func TestValidateEdgeDocument(t *testing.T) {
	tests := []struct {
		name          string
		document      any
		expectedError error
	}{
		{
			name: "Success - Valid Edge Document",
			document: map[string]any{
				"_from": "vertex1",
				"_to":   "vertex2",
			},
			expectedError: nil,
		},
		{
			name:          "Fail - Document is Not a Map",
			document:      "invalid",
			expectedError: errInvalidEdgeDocumentType,
		},
		{
			name: "Fail - Missing _from Field",
			document: map[string]any{
				"_to": "vertex2",
			},
			expectedError: errMissingEdgeFields,
		},
		{
			name: "Fail - Missing _to Field",
			document: map[string]any{
				"_from": "vertex1",
			},
			expectedError: errMissingEdgeFields,
		},
		{
			name: "Fail - _from is Not a String",
			document: map[string]any{
				"_from": 123,
				"_to":   "vertex2",
			},
			expectedError: errInvalidFromField,
		},
		{
			name: "Fail - _to is Not a String",
			document: map[string]any{
				"_from": "vertex1",
				"_to":   123,
			},
			expectedError: errInvalidToField,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEdgeDocument(tc.document)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
