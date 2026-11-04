package handler

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestStatusForKind(t *testing.T) {
	require.Equal(t, http.StatusBadRequest, statusForKind(domain.KindValidation))
	require.Equal(t, http.StatusConflict, statusForKind(domain.KindConflict))
	require.Equal(t, http.StatusNotFound, statusForKind(domain.KindNotFound))
	require.Equal(t, http.StatusForbidden, statusForKind(domain.KindForbidden))
	require.Equal(t, http.StatusUnauthorized, statusForKind(domain.KindUnauthorized))
	require.Equal(t, http.StatusInternalServerError, statusForKind(domain.Kind(99)))
}

func TestHandleError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"Validation", domain.NewValidationError("X", "inválido"), http.StatusBadRequest, `{"error":"inválido","code":"X"}`},
		{"Conflict", domain.NewConflictError("C", "conflito"), http.StatusConflict, `{"error":"conflito","code":"C"}`},
		{"NotFound", domain.NewNotFoundError("N", "não achou"), http.StatusNotFound, `{"error":"não achou","code":"N"}`},
		{"Forbidden", domain.NewForbiddenError("F", "proibido"), http.StatusForbidden, `{"error":"proibido","code":"F"}`},
		{"Unauthorized", domain.NewUnauthorizedError("U", "sem acesso"), http.StatusUnauthorized, `{"error":"sem acesso","code":"U"}`},
		{"WrappedDomain", fmt.Errorf("%w: Filtro", domain.NewConflictError("E", "estoque")), http.StatusConflict, `{"error":"estoque: Filtro","code":"E"}`},
		{"NoRows", sql.ErrNoRows, http.StatusNotFound, `{"error":"recurso não encontrado","code":"NAO_ENCONTRADO"}`},
		{"Internal", errors.New("pq: password authentication failed"), http.StatusInternalServerError, `{"error":"erro interno do servidor","code":"ERRO_INTERNO"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

			handleError(ctx, tc.err)

			require.Equal(t, tc.status, recorder.Code)
			require.JSONEq(t, tc.body, recorder.Body.String())
		})
	}
}

func TestErrorResponse(t *testing.T) {
	require.Equal(t, gin.H{"error": "x"}, errorResponse(errors.New("x")))
}

func TestBindErrorTraduzValidacao(t *testing.T) {
	type req struct {
		Email string `json:"email" binding:"required,email"`
		Senha string `json:"senha" binding:"required,min=8"`
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"senha":"123"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var body req
	err := ctx.ShouldBindJSON(&body)
	require.Error(t, err)
	bindError(ctx, err)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"error":"campo 'email' é obrigatório","code":"REQUISICAO_INVALIDA","campos":[{"campo":"email","regra":"required"},{"campo":"senha","regra":"min"}]}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "req.")
}

func TestBindErrorJSONMalformado(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/x", nil)

	bindError(ctx, errors.New("unexpected EOF"))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"error":"requisição inválida: corpo ou parâmetros malformados","code":"REQUISICAO_INVALIDA"}`, recorder.Body.String())
}

func TestBindErrorCorpoGrande(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/x", nil)

	bindError(ctx, &http.MaxBytesError{Limit: 1})
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), "CORPO_GRANDE_DEMAIS")
}

func TestDescreverCampo(t *testing.T) {
	type req struct {
		Email  string `json:"email" binding:"email"`
		Papel  string `json:"papel" binding:"oneof=A B"`
		Nome   string `json:"nome" binding:"max=2"`
		Ano    int    `json:"ano" binding:"gte=1900"`
		Codigo string `json:"codigo" binding:"alphanum"`
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"email":"x","papel":"C","nome":"abc","ano":1,"codigo":"!"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	var body req
	err := ctx.ShouldBindJSON(&body)
	require.Error(t, err)

	var verrs validator.ValidationErrors
	require.ErrorAs(t, err, &verrs)
	mensagens := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		mensagens = append(mensagens, descreverCampo(fe))
	}
	require.Equal(t, []string{
		"campo 'email' deve ser um e-mail válido",
		"campo 'papel' deve ser um de: A B",
		"campo 'nome' deve ter no máximo 2",
		"campo 'ano' fora do intervalo permitido (gte 1900)",
		"campo 'codigo' inválido (regra alphanum)",
	}, mensagens)
}
