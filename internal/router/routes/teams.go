package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"plassstic.tech/trainee/avito/internal/schema"
	"plassstic.tech/trainee/avito/internal/service"
)

func SetupTeamRoutes(team *gin.RouterGroup, service service.Service) {
	team.POST("/add", addTeam(service))
	team.GET("/get", getTeam(service))
}

func addTeam(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var team schema.Team
		if err := c.ShouldBindJSON(&team); err != nil {
			respondError(c, schema.Err{}.Wrap(schema.Unknown, err))
			return
		}

		result, serr := service.AddTeam(c, team)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusCreated, schema.AddTeamResponse{Team: *result})
	}
}

func getTeam(service service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamName := c.Query("team_name")
		if teamName == "" {
			respondError(c, &schema.Err{Code: schema.Unknown, Msg: "team_name is required"})
			return
		}

		result, serr := service.GetTeam(c, teamName)
		if serr != nil {
			respondError(c, serr)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
