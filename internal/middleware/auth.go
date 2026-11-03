package middleware

import (
	"errors"
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

// Auth exige um access token Bearer válido e carrega o usuário na requisição.
// Refresh tokens são recusados: eles só servem em POST /tokens/refresh.
func Auth(tokenMaker token.Maker, store db.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			abortarNaoAutorizado(ctx, "TOKEN_AUSENTE", "o cabeçalho Authorization é obrigatório")
			return
		}
		fields := strings.Fields(authorizationHeader)
		if len(fields) != 2 || strings.ToLower(fields[0]) != AuthorizationTypeBearer {
			abortarNaoAutorizado(ctx, "TOKEN_FORMATO_INVALIDO", "use o formato: Authorization: Bearer <access_token>")
			return
		}

		payload, err := tokenMaker.VerifyToken(fields[1])
		if err != nil {
			if errors.Is(err, token.ErrExpiredToken) {
				abortarNaoAutorizado(ctx, "TOKEN_EXPIRADO", "token expirado, faça login de novo ou renove o access token")
				return
			}
			abortarNaoAutorizado(ctx, "TOKEN_INVALIDO", "token inválido")
			return
		}
		if !payload.EhDoTipo(token.TipoAccess) {
			abortarNaoAutorizado(ctx, "TOKEN_TIPO_INVALIDO", "use o access token; o refresh token só renova a sessão em POST /tokens/refresh")
			return
		}

		usuario, err := store.GetUsuario(ctx.Request.Context(), payload.UserUUID)
		if err != nil {
			abortarNaoAutorizado(ctx, "USUARIO_NAO_ENCONTRADO", "usuário do token não encontrado")
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
			abortarProibido(ctx)
			return
		}

		usuario, ok := value.(db.Usuario)
		if !ok {
			abortarProibido(ctx)
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

		abortarProibido(ctx)
	}
}

func abortarNaoAutorizado(ctx *gin.Context, code, message string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": code})
}

func abortarProibido(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": "o seu papel não tem permissão para esta operação",
		"code":  "PAPEL_SEM_PERMISSAO",
	})
}
