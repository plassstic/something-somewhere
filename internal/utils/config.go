package utils

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type PgConfig struct {
	User     string `env:"USER,required,notEmpty"`
	Password string `env:"PASSWORD,required,notEmpty"`
	Host     string `env:"HOST,notEmpty" envDefault:"localhost"`
	Port     string `env:"PORT,notEmpty" envDefault:"5432"`
	Db       string `env:"DB"`
}

func (p PgConfig) URL() string {
	if p.Db == "" {
		p.Db = p.User
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.Db,
	)
}

type Server struct {
	Port int `env:"PORT" envDefault:"8080"`
}

type Config struct {
	PgConfig `envPrefix:"POSTGRES_"`
	Server   `envPrefix:"SERVER_"`
}

func (c Config) PostgresURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PgConfig.User,
		c.PgConfig.Password,
		c.PgConfig.Host,
		c.PgConfig.Port,
		c.PgConfig.User,
	)
}

func ParseConfig() *Config {
	cfg := env.Must(env.ParseAs[Config]())
	return &cfg
}
