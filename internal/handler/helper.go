package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetHealth godoc
// @Summary      Verificar disponibilidade da API (health check)
// @Description  Responde 200 quando a API está no ar. Usado pelo healthcheck do docker-compose e por orquestradores.
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]string "Sucesso"
// @Router       /health [get]
func (server *Server) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Oficina API is running",
	})
}

// GetHelper godoc
// @Summary      Verificar disponibilidade da API (obsoleto, use /health)
// @Description  Mantido por compatibilidade. Use GET /health.
// @Tags         health
// @Produce      json
// @Deprecated
// @Success      200 {object} map[string]string "Sucesso"
// @Router       /helper [get]
func (server *Server) GetHelper(ctx *gin.Context) {
	server.GetHealth(ctx)
}
