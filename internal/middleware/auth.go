package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey  = "authorization"
	AuthorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
	CurrentUserKey          = "current_user"

	PapelAdministrador = "ADMINISTRADOR"
	PapelAtendente     = "ATENDENTE"
	PapelMecanico      = "MECANICO"
)

func Auth(tokenMaker token.Maker, store db.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != AuthorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		usuario, err := store.GetUsuario(context.Background(), payload.UserUUID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("user not found")))
			return
		}

		if !usuario.Ativo {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "usuário inativo",
				"code":  "USUARIO_INATIVO",
			})
			return
		}

		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Set(CurrentUserKey, usuario)
		ctx.Next()
	}
}

// RequireRole exige que o usuário autenticado (já carregado pelo Auth, que
// roda antes na mesma cadeia) tenha um dos papéis informados. O
// administrador sempre passa.
func RequireRole(papeis ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		value, exists := ctx.Get(CurrentUserKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(errors.New("forbidden")))
			return
		}

		usuario, ok := value.(db.Usuario)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(errors.New("forbidden")))
			return
		}

		if usuario.Papel == PapelAdministrador {
			ctx.Next()
			return
		}
		for _, papel := range papeis {
			if usuario.Papel == papel {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "o seu papel não tem permissão para esta operação",
			"code":  "PAPEL_SEM_PERMISSAO",
		})
	}
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
