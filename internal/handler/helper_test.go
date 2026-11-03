package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	"github.com/stretchr/testify/require"
)

// /health é a rota de disponibilidade; /helper continua por compatibilidade.
func TestGetHealthAPI(t *testing.T) {
	store := mockdb.NewMockStore(nil)
	server := newTestServer(t, store)

	for _, path := range []string{"/api/v1/health", "/api/v1/helper"} {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		server.router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code, path)
		require.JSONEq(t, `{"status":"ok","message":"Oficina API is running"}`, recorder.Body.String())
	}
}
