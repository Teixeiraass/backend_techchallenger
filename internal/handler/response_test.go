package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/gin-gonic/gin"
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
