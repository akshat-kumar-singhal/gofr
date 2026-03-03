package arangodb

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gofr.dev/pkg/gofr/datasource/arangodb/mocks"
)

func TestClient_CreateUser(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		username      string
		userOpts      UserOptions
		setupMocks    func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "createUser",
			username:   "test",
			userOpts: UserOptions{
				Password: "user123",
				Extra:    nil,
			},
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().CreateUser(gomock.Any(), "test", gomock.Any()).Return(nil, nil)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			tc.setupMocks(ctrl, mockArango)

			err := client.createUser(context.Background(), tc.username, tc.userOpts)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_DropUser(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		username      string
		setupMocks    func(ctrl *gomock.Controller, mockArango *mocks.MockClient)
		expectedError error
	}{
		{
			name:       "Success",
			expectedOp: "dropUser",
			username:   "test",
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient) {
				mockArango.EXPECT().RemoveUser(gomock.Any(), "test").Return(nil)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			tc.setupMocks(ctrl, mockArango)

			err := client.dropUser(context.Background(), tc.username)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_GrantDB(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOp    string
		dbName        string
		username      string
		permission    string
		setupMocks    func(ctrl *gomock.Controller, mockArango *mocks.MockClient, mockUser *mocks.MockUser)
		expectedError error
	}{
		{
			name:       "Success_ReadWrite",
			expectedOp: "grantDB",
			dbName:     "testDB",
			username:   "testUser",
			permission: string(arangodb.GrantReadWrite),
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient, mockUser *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(mockUser, nil)
			},
			expectedError: nil,
		},
		{
			name:       "Success_ReadOnly",
			expectedOp: "grantDB",
			dbName:     "testDB",
			username:   "testUser",
			permission: string(arangodb.GrantReadOnly),
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient, mockUser *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(mockUser, nil)
			},
			expectedError: nil,
		},
		{
			name:       "Error_UserNotFound",
			expectedOp: "grantDB",
			dbName:     "testDB",
			username:   "testUser",
			permission: string(arangodb.GrantReadWrite),
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient, _ *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(nil, errUserNotFound)
			},
			expectedError: errUserNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			mockUser := mocks.NewMockUser(ctrl)

			tc.setupMocks(ctrl, mockArango, mockUser)

			err := client.grantDB(context.Background(), tc.dbName, tc.username, tc.permission)

			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_GrantCollection(t *testing.T) {
	testCases := []struct {
		name           string
		expectedOp     string
		dbName         string
		collectionName string
		username       string
		permission     string
		setupMocks     func(ctrl *gomock.Controller, mockArango *mocks.MockClient, mockUser *mocks.MockUser)
		expectedError  error
	}{
		{
			name:           "Success",
			expectedOp:     "grantCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			username:       "testUser",
			permission:     string(arangodb.GrantReadOnly),
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient, mockUser *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(mockUser, nil)
			},
			expectedError: nil,
		},
		{
			name:           "Error_UserNotFound",
			expectedOp:     "grantCollection",
			dbName:         "testDB",
			collectionName: "testCollection",
			username:       "testUser",
			permission:     string(arangodb.GrantReadOnly),
			setupMocks: func(_ *gomock.Controller, mockArango *mocks.MockClient, _ *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(nil, errUserNotFound)
			},
			expectedError: errUserNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mockArango, ctrl := setupTestClient(t, tc.expectedOp)
			defer ctrl.Finish()

			mockUser := mocks.NewMockUser(ctrl)

			tc.setupMocks(ctrl, mockArango, mockUser)

			err := client.grantCollection(context.Background(), tc.dbName, tc.collectionName, tc.username, tc.permission)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUser(t *testing.T) {
	testCases := []struct {
		name          string
		username      string
		setupMocks    func(mockArango *mocks.MockClient, mockUser *mocks.MockUser)
		expectUser    bool
		expectedError error
	}{
		{
			name:     "Success",
			username: "testUser",
			setupMocks: func(mockArango *mocks.MockClient, mockUser *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(mockUser, nil)
			},
			expectUser:    true,
			expectedError: nil,
		},
		{
			name:     "Error_UserNotFound",
			username: "testUser",
			setupMocks: func(mockArango *mocks.MockClient, _ *mocks.MockUser) {
				mockArango.EXPECT().User(gomock.Any(), "testUser").Return(nil, errUserNotFound)
			},
			expectUser:    false,
			expectedError: errUserNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockUser := mocks.NewMockUser(ctrl)
			client := &Client{client: mockArango}

			tc.setupMocks(mockArango, mockUser)

			user, err := client.user(context.Background(), tc.username)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
			}
		})
	}
}

func TestClient_Database(t *testing.T) {
	testCases := []struct {
		name          string
		dbName        string
		setupMocks    func(mockArango *mocks.MockClient, mockDB *mocks.MockDatabase)
		validate      func(t *testing.T, db arangodb.Database, mockDB *mocks.MockDatabase)
		expectedError error
	}{
		{
			name:   "Success",
			dbName: "testDB",
			setupMocks: func(mockArango *mocks.MockClient, mockDB *mocks.MockDatabase) {
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().Name().Return("testDB")
			},
			validate: func(t *testing.T, db arangodb.Database, _ *mocks.MockDatabase) {
				t.Helper()
				require.NotNil(t, db)
				require.Equal(t, "testDB", db.Name())
			},
			expectedError: nil,
		},
		{
			name:   "Error_DBNotFound",
			dbName: "testDB",
			setupMocks: func(mockArango *mocks.MockClient, _ *mocks.MockDatabase) {
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(nil, errDBNotFound)
			},
			validate: func(t *testing.T, db arangodb.Database, _ *mocks.MockDatabase) {
				t.Helper()
				require.Nil(t, db)
			},
			expectedError: errDBNotFound,
		},
		{
			name:   "DatabaseOperations",
			dbName: "testDB",
			setupMocks: func(mockArango *mocks.MockClient, mockDB *mocks.MockDatabase) {
				mockArango.EXPECT().GetDatabase(gomock.Any(), "testDB", nil).Return(mockDB, nil)
				mockDB.EXPECT().Name().Return("testDB")
				mockDB.EXPECT().Remove(gomock.Any()).Return(nil)
				mockDB.EXPECT().GetCollection(gomock.Any(), "testCollection", nil).Return(nil, nil)
			},
			validate: func(t *testing.T, db arangodb.Database, _ *mocks.MockDatabase) {
				t.Helper()
				require.NotNil(t, db)
				require.Equal(t, "testDB", db.Name())

				err := db.Remove(context.Background())
				require.NoError(t, err)

				coll, err := db.GetCollection(context.Background(), "testCollection", nil)
				require.NoError(t, err)
				require.Nil(t, coll)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockArango := mocks.NewMockClient(ctrl)
			mockDB := mocks.NewMockDatabase(ctrl)

			client := New(Config{Host: "localhost", Port: 8527, User: "root", Password: "root"})
			client.client = mockArango

			tc.setupMocks(mockArango, mockDB)

			db, err := client.database(context.Background(), tc.dbName)

			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tc.validate(t, db, mockDB)
		})
	}
}
