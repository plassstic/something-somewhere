package utils

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type box struct {
	ctx context.Context

	dbpool *pgxpool.Pool
}

type Box interface {
	Pg() *pgxpool.Pool
}

func (b *box) setupPg(cfg PgConfig) {
	var err error
	b.dbpool, err = pgxpool.New(b.ctx, cfg.URL())
	if err != nil {
		log.Fatal().AnErr("err", err).Msg("failed to connect to database")
	} else if b.dbpool.Ping(b.ctx) != nil {
		log.Fatal().AnErr("err", err).Msg("failed to ping database")
	}
}

func SetupBox(ctx context.Context, cfg *Config) Box {
	b := box{ctx: ctx}
	b.setupPg(cfg.PgConfig)
	return &b
}

func (b box) Pg() *pgxpool.Pool {
	return b.dbpool
}
