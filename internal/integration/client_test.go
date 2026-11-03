package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	emailAdmin     = "admin@oficina.com"
	senhaAdmin     = "admin123"
	emailAtendente = "atendente@oficina.com"
	senhaAtendente = "atendente123"
	emailMecanico  = "mecanico@oficina.com"
	senhaMecanico  = "mecanico123"
)

type apiResponse struct {
	Code int
	Body []byte
}

func (r apiResponse) decode(t *testing.T, target any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(r.Body, target), string(r.Body))
}

func call(t *testing.T, method, path, token string, body any) apiResponse {
	t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, "/api/v1"+path, reader)
	require.NoError(t, err)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	testRouter.ServeHTTP(recorder, request)
	return apiResponse{Code: recorder.Code, Body: recorder.Body.Bytes()}
}

func login(t *testing.T, email, senha string) string {
	t.Helper()

	rsp := call(t, http.MethodPost, "/usuarios/login", "", map[string]string{"email": email, "senha": senha})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	var body struct {
		AccessToken string `json:"access_token"`
	}
	rsp.decode(t, &body)
	// JWT: cabeçalho.payload.assinatura.
	require.Len(t, strings.Split(body.AccessToken, "."), 3)
	return body.AccessToken
}
