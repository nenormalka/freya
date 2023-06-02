package http

import (
	"time"

	"github.com/nenormalka/freya/config"
)

type Config struct {
	ListenAddr       string
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
	ReleaseID        string
}

func NewHTTPConfig(cfg *config.Config) Config {
	return Config{
		ListenAddr:       cfg.HTTP.ListenAddr,
		KeepaliveTime:    cfg.HTTP.KeepaliveTime,
		KeepaliveTimeout: cfg.HTTP.KeepaliveTimeout,
		ReleaseID:        cfg.ReleaseID,
	}
}
