package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	"github.com/stretchr/testify/require"
)

func TestGetHelperAPI(t *testing.T) {
	store := mockdb.NewMockStore(nil)
	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()

	request, err := http.NewRequest(http.MethodGet, "/api/v1/helper", nil)
	require.NoError(t, err)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"status":"ok","message":"Oficina API is running"}`, recorder.Body.String())
}
