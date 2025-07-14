package model

import "github.com/nenormalka/freya/config"

type (
	Config struct {
		ServiceName   string
		TransportType string
		GRPCAddresses string
		KafkaTopic    string
		Enabled       bool
	}
)

func NewConfig(cfg *config.Config) *Config {
	return &Config{
		ServiceName:   cfg.AppName,
		TransportType: cfg.Communication.TransportType,
		GRPCAddresses: cfg.Communication.GRPCAddresses,
		KafkaTopic:    cfg.Communication.KafkaTopic,
		Enabled:       cfg.Communication.Enabled,
	}
}
