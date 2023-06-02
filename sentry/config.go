package sentry

import "github.com/nenormalka/freya/config"

type (
	Config struct {
		DSN         string
		Environment string
		ReleaseID   string
	}
)

func NewSentryConfig(cfg *config.Config) Config {
	return Config{
		DSN:         cfg.Sentry.DSN,
		Environment: cfg.Env,
		ReleaseID:   cfg.AppName + "@" + cfg.ReleaseID,
	}
}
