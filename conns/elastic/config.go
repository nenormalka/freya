package elastic

import (
	"github.com/nenormalka/freya/config"
)

type Config struct {
	DSN        string
	MaxRetries int
	WithLogger bool
}

func NewConfig(cfg *config.Config) Config {
	return Config{
		DSN:        cfg.ElasticSearch.DSN,
		MaxRetries: cfg.ElasticSearch.MaxRetries,
		WithLogger: cfg.ElasticSearch.WithLogger,
	}
}
