package config

import (
	"time"

	"github.com/nenormalka/freya/config"
)

type (
	Config struct {
		Address            string
		Scheme             string
		Token              string
		ServiceName        string
		SessionTTL         string
		LeaderTTL          time.Duration
		InsecureSkipVerify bool
	}
)

func CreateConfig(cfg *config.Config) Config {
	return Config{
		Address:            cfg.ConsulConfig.Address,
		Scheme:             cfg.ConsulConfig.Scheme,
		Token:              cfg.ConsulConfig.Token,
		SessionTTL:         cfg.ConsulConfig.SessionTTL,
		InsecureSkipVerify: cfg.ConsulConfig.InsecureSkipVerify,
		LeaderTTL:          cfg.ConsulConfig.LeaderTTL,
		ServiceName:        cfg.AppName,
	}
}
