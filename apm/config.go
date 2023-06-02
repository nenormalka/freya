package apm

import "github.com/nenormalka/freya/config"

type (
	Config struct {
		ServiceName string
		ServerURL   string
		Environment string
		ReleaseID   string
	}
)

func NewAPMConfig(cfg *config.Config) Config {
	return Config{
		ServiceName: cfg.APM.ServiceName,
		ServerURL:   cfg.APM.ServerURL,
		Environment: cfg.APM.Environment,
		ReleaseID:   cfg.ReleaseID,
	}
}
