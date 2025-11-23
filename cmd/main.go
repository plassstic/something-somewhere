package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"plassstic.tech/trainee/avito/internal/router"
	"plassstic.tech/trainee/avito/internal/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	utils.SetupLogging()
	cfg := utils.ParseConfig()

	router.New(utils.SetupBox(ctx, cfg)).Serve(ctx, cfg.Server.Port)
}
