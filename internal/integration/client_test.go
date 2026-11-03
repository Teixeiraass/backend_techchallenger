package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	return callCom(t, testRouter, method, path, token, body, nil)
}

// callDocumento envia o CPF/CNPJ no header X-Documento.
func callDocumento(t *testing.T, method, path, token, documento string, body any) apiResponse {
	t.Helper()
	return callCom(t, testRouter, method, path, token, body, map[string]string{"X-Documento": documento})
}

func callCom(t *testing.T, router *gin.Engine, method, path, token string, body any, headers map[string]string) apiResponse {
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
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
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
	require.NotEmpty(t, body.AccessToken)
	return body.AccessToken
}
