package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/middleware"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorResponse is the body returned by every error.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// handleError maps domain errors to HTTP status codes. Unexpected errors are
// logged and answered with a generic message so internals do not leak.
func handleError(ctx *gin.Context, err error) {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		ctx.JSON(statusForKind(domainErr.Kind), ErrorResponse{Error: err.Error(), Code: domainErr.Code})
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Error: "recurso não encontrado", Code: "NAO_ENCONTRADO"})
		return
	}

	log.Printf("internal error on %s %s: %v", ctx.Request.Method, ctx.FullPath(), err)
	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: "erro interno do servidor", Code: "ERRO_INTERNO"})
}

func statusForKind(kind domain.Kind) int {
	switch kind {
	case domain.KindValidation:
		return http.StatusBadRequest
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindForbidden:
		return http.StatusForbidden
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func bindError(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: "REQUISICAO_INVALIDA"})
}

// uuidParam reads a UUID path param, answering 400 when it is malformed.
func uuidParam(ctx *gin.Context, name string) (uuid.UUID, bool) {
	value, err := uuid.Parse(ctx.Param(name))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "parâmetro " + name + " deve ser um UUID válido", Code: "UUID_INVALIDO"})
		return uuid.Nil, false
	}
	return value, true
}

func currentPayload(ctx *gin.Context) *token.Payload {
	return ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
}

func currentUserID(ctx *gin.Context) int64 {
	return currentPayload(ctx).UserID
}
