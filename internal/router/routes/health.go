package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"plassstic.tech/trainee/avito/internal/schema"
)

func SetupHealthRoute(r *gin.Engine) {
	r.GET("/health", health())
}

func health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, schema.HealthResponse{Status: "ok"})
	}
}
