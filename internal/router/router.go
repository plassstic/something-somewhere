package router

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"plassstic.tech/trainee/avito/internal/router/routes"
	"plassstic.tech/trainee/avito/internal/service"
	"plassstic.tech/trainee/avito/internal/utils"
)

type router struct {
	*gin.Engine
	service service.Service
}

func (r *router) Serve(ctx context.Context, port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal().Int("port", port).Err(err).Msg("failed to listen")
	}

	log.Info().Int("port", port).Msg("OK, registered")
	srv := http.Server{Handler: r.Handler()}

	go func() {
		<-ctx.Done()
		log.Info().Msg("context done, closing listener")
		_ = lis.Close()
		tctx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(tctx)
		log.Info().Msg("OK, closed")
	}()

	if err = srv.Serve(lis); err != nil {
		log.Fatal().Int("port", port).Err(err).Msg("failed to serve")
	}
}

type Router interface {
	Serve(ctx context.Context, port int)
}

func New(box utils.Box) Router {
	r := &router{
		Engine:  gin.New(),
		service: service.New(box.Pg()),
	}
	gin.DefaultWriter = log.Logger
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(r.errorHandler)
	gin.SetMode(gin.ReleaseMode)
	r.setupRoutes()
	return r
}

func (r *router) errorHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) > 0 {
		log.Error().Err(c.Errors.Last()).Msg("handler error")
	}
}

func (r *router) setupRoutes() {
	routes.SetupTeamRoutes(r.Engine.Group("/team"), r.service)
	routes.SetupUsersRoutes(r.Engine.Group("/users"), r.service)
	routes.SetupPRRoutes(r.Engine.Group("/pullRequest"), r.service)
	routes.SetupHealthRoute(r.Engine)
}
