package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LimiteCorpo limita o tamanho do corpo de toda requisição. Corpo maior que
// maxBytes faz a leitura falhar e o handler responde 413 (ver handler.bindError).
func LimiteCorpo(maxBytes int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Body != nil {
			ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)
		}
		ctx.Next()
	}
}
