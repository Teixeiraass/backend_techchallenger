package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetHelper godoc
// @Summary      Verificar disponibilidade da API
// @Description  Endpoint simples para confirmar que a API está respondendo.
// @Tags         helper
// @Produce      json
// @Success      200 {object} map[string]string "Sucesso"
// @Router       /helper [get]
func (server *Server) GetHelper(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Oficina API is running",
	})
}
