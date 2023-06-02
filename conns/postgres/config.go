package postrgres

import (
	"time"

	"github.com/nenormalka/freya/config"
)

type (
	PostgresConfig struct {
		DSN                string
		MaxOpenConnections int
		MaxIdleConnections int
		ConnMaxLifetime    time.Duration
	}
)

func NewPostgresConfig(cfg *config.Config) PostgresConfig {
	return PostgresConfig{
		DSN:                cfg.Postgres.DSN,
		MaxOpenConnections: cfg.Postgres.MaxOpenConnections,
		MaxIdleConnections: cfg.Postgres.MaxIdleConnections,
		ConnMaxLifetime:    cfg.Postgres.ConnMaxLifetime,
	}
}
