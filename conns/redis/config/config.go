package config

import (
	lilith "github.com/nenormalka/lilith/methods"

	"github.com/nenormalka/freya/config"
)

type (
	Config struct {
		DSN      string
		PoolSize int
	}
)

func CreateConfig(cfg *config.Config) *Config {
	return &Config{
		DSN:      lilith.Ternary(cfg.RedisConfig.RedisDSN != "", cfg.RedisConfig.RedisDSN, cfg.RedisConfig.KeyDBDSN),
		PoolSize: cfg.RedisConfig.PoolSize,
	}
}
