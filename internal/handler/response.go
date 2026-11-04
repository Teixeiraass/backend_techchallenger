package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/middleware"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ErrorResponse is the body returned by every error.
type ErrorResponse struct {
	Error  string        `json:"error"`
	Code   string        `json:"code,omitempty"`
	Campos []CampoErrado `json:"campos,omitempty"`
}

// CampoErrado identifica um campo do corpo (nome JSON) que falhou na validação.
type CampoErrado struct {
	Campo string `json:"campo"`
	Regra string `json:"regra"`
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

// bindError responde 400 para corpo ou query inválidos. Erros do validador
// são traduzidos para o nome JSON do campo, sem expor nomes de struct Go.
func bindError(ctx *gin.Context, err error) {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		ctx.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "corpo da requisição grande demais", Code: "CORPO_GRANDE_DEMAIS"})
		return
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		campos := make([]CampoErrado, 0, len(validationErrs))
		for _, fe := range validationErrs {
			campos = append(campos, CampoErrado{Campo: nomeJSON(fe), Regra: fe.Tag()})
		}
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: descreverCampo(validationErrs[0]), Code: "REQUISICAO_INVALIDA", Campos: campos})
		return
	}

	ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "requisição inválida: corpo ou parâmetros malformados", Code: "REQUISICAO_INVALIDA"})
}

func init() {
	// Faz o validador reportar o nome JSON (ou da query) do campo em vez do
	// nome do campo da struct Go.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			for _, tag := range []string{"json", "form"} {
				nome := strings.SplitN(fld.Tag.Get(tag), ",", 2)[0]
				if nome != "" && nome != "-" {
					return nome
				}
			}
			return fld.Name
		})
	}
}

// nomeJSON devolve o caminho do campo como aparece no JSON/query
// (ex.: "email" ou "pecas[0].quantidade"), sem o nome da struct de entrada.
func nomeJSON(fe validator.FieldError) string {
	ns := fe.Namespace()
	if i := strings.Index(ns, "."); i >= 0 {
		return ns[i+1:]
	}
	return fe.Field()
}

func descreverCampo(fe validator.FieldError) string {
	campo := nomeJSON(fe)
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("campo '%s' é obrigatório", campo)
	case "email":
		return fmt.Sprintf("campo '%s' deve ser um e-mail válido", campo)
	case "min":
		return fmt.Sprintf("campo '%s' deve ter no mínimo %s", campo, fe.Param())
	case "max":
		return fmt.Sprintf("campo '%s' deve ter no máximo %s", campo, fe.Param())
	case "oneof":
		return fmt.Sprintf("campo '%s' deve ser um de: %s", campo, fe.Param())
	case "gt", "gte", "lt", "lte":
		return fmt.Sprintf("campo '%s' fora do intervalo permitido (%s %s)", campo, fe.Tag(), fe.Param())
	default:
		return fmt.Sprintf("campo '%s' inválido (regra %s)", campo, fe.Tag())
	}
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
