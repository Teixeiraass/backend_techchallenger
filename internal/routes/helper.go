package routes

import "github.com/gin-gonic/gin"

type HelperHandler interface {
	GetHealth(c *gin.Context)
	GetHelper(c *gin.Context)
}

func registerHelperRoutes(router gin.IRoutes, h HelperHandler) {
	router.GET("/health", h.GetHealth)
	// Obsoleto: mantido por compatibilidade, use /health.
	router.GET("/helper", h.GetHelper)
}
