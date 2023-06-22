package postrgres

import (
	"time"

	"github.com/nenormalka/freya/config"
)

type (
	PostgresConfig struct {
		Configs []DBConfig
	}

	DBConfig struct {
		DSN  string
		Name string
		// Type pgx|sqlx
		Type               string
		MaxOpenConnections int
		MaxIdleConnections int
		ConnMaxLifetime    time.Duration
	}
)

func NewPostgresConfig(cfg *config.Config) PostgresConfig {
	if len(cfg.DB) == 0 {
		return PostgresConfig{}
	}

	cfgs := make([]DBConfig, len(cfg.DB))

	for i := range cfg.DB {
		cfgs[i] = DBConfig{
			DSN:                cfg.DB[i].DSN,
			Name:               cfg.DB[i].Name,
			MaxOpenConnections: cfg.DB[i].MaxOpenConnections,
			MaxIdleConnections: cfg.DB[i].MaxIdleConnections,
			ConnMaxLifetime:    cfg.DB[i].ConnMaxLifetime,
			Type:               cfg.DB[i].Type,
		}
	}

	return PostgresConfig{
		Configs: cfgs,
	}
}
