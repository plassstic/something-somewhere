package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"plassstic.tech/trainee/avito/internal/schema"
	"plassstic.tech/trainee/avito/internal/service"
)

func SetupUsersRoutes(users *gin.RouterGroup, service service.Service) {
	users.POST("/setIsActive", setUserActive(service))
	users.GET("/getReview", getUserReviews(service))
}

func setUserActive(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schema.SetUserActiveRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, schema.Err{}.Wrap(schema.Unknown, err))
			return
		}

		result, serr := service.SetUserActive(c, req.UserID, req.IsActive)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusOK, schema.UserResponse{User: *result})
	}
}

func getUserReviews(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			respondError(c, &schema.Err{Code: schema.Unknown, Msg: "user_id is required"})
			return
		}

		prs, serr := service.GetUserReviews(c, userID)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusOK, schema.UserReviewsResponse{
			UserID:       userID,
			PullRequests: prs,
		})
	}
}
