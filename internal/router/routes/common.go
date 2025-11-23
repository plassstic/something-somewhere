package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"plassstic.tech/trainee/avito/internal/schema"
)

func respondError(c *gin.Context, err *schema.Err) {
	var status int
	switch err.Code {
	case schema.TeamExists:
		status = http.StatusBadRequest
	case schema.PRExists:
		status = http.StatusConflict
	case schema.PRMerged, schema.NotAssigned, schema.NoCandidate, schema.NotFound:
		status = http.StatusNotFound
	default:
		status = http.StatusInternalServerError
	}

	c.JSON(status, schema.ErrorResponse{
		Err: *err,
	})
}
