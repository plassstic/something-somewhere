package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"plassstic.tech/trainee/avito/internal/schema"
	"plassstic.tech/trainee/avito/internal/service"
)

func SetupPRRoutes(pr *gin.RouterGroup, service service.Service) {
	pr.POST("/create", createPR(service))
	pr.POST("/merge", mergePR(service))
	pr.POST("/reassign", reassignReviewer(service))
}

func createPR(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schema.CreatePRRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, schema.Err{}.Wrap(schema.Unknown, err))
			return
		}

		result, serr := service.CreatePR(c, req)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusCreated, schema.PRResponse{PR: *result})
	}
}

func mergePR(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schema.MergePRRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, schema.Err{}.Wrap(schema.Unknown, err))
			return
		}

		result, serr := service.MergePR(c, req.PRId)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusOK, schema.PRResponse{PR: *result})
	}
}

func reassignReviewer(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schema.ReassignReviewerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, schema.Err{}.Wrap(schema.Unknown, err))
			return
		}

		newID, pr, serr := service.ReassignReviewer(c, req.PRId, req.OldUserID)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusOK, schema.ReassignResponse{
			PR:      *pr,
			NewUser: newID,
		})
	}
}
