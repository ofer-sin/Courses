package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/ofer-sin/Courses/BackendCourse/simplebank/db/mock"
	db "github.com/ofer-sin/Courses/BackendCourse/simplebank/db/sqlc"
	"github.com/ofer-sin/Courses/BackendCourse/simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestGetAccountAPI(t *testing.T) {
	account := randomAccount()

	testCases := []struct {
		name          string
		accountId     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				// Set up the expected call to the mock store GetAccount() method
				// with the context and the account ID
				// and specify that it should be called once
				// and return the account and nil error
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				// return an empty account and ErrNoRows error
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Accounts{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Accounts{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			accountId: 0, // invalid ID
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0) //The GetAccount() method should not be called
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// loop over the test cases
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t) //required for mocking

			// This line is not needed here, as we are using the controller in the test function
			// It will be called at the end of the test function
			// to clean up the mock controller
			// and release any resources it holds.
			// This is a common pattern in Go testing with gomock
			defer ctrl.Finish()

			// Create a mock store (instead of the real store)
			store := mockdb.NewMockStore(ctrl)

			// Call the buildStubs function to set up the expected calls
			tc.buildStubs(store)

			// Create a new server with the mock store
			server := NewServer(store)

			// Create a new HTTP request to the /accounts/:id endpoint
			// Create a new HTTP recorder to capture the response
			recorder := httptest.NewRecorder()
			url := fmt.Sprintf("/accounts/%d", tc.accountId)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Serve the HTTP request using the test server
			// This will call the getAccount() method of the server
			// which will in turn call the GetAccount() method of the mock store
			server.router.ServeHTTP(recorder, request)

			// Check the response using the checkResponse function
			tc.checkResponse(t, recorder)
		})
	}
}

func randomAccount() db.Accounts {
	return db.Accounts{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, account db.Accounts) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Accounts
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}
