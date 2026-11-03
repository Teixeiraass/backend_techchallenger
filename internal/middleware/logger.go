package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger registra método, caminho (sem query string), status e latência de
// cada requisição. A query string fica de fora de propósito: ela pode levar
// dado pessoal (CPF/CNPJ) e dado pessoal não vai para o log (LGPD).
func Logger() gin.HandlerFunc {
	return LoggerCom(log.Printf)
}

// LoggerCom permite trocar o destino do log (usado nos testes).
func LoggerCom(printf func(format string, args ...any)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		inicio := time.Now()
		caminho := ctx.Request.URL.Path

		ctx.Next()

		printf("%s %s -> %d (%s)", ctx.Request.Method, caminho, ctx.Writer.Status(), time.Since(inicio).Round(time.Millisecond))
	}
}
