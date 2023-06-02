package grpc

import (
	"time"

	"github.com/nenormalka/freya/config"
)

type (
	Config struct {
		ListenAddr       string
		WithReflection   bool
		KeepaliveTime    time.Duration
		KeepaliveTimeout time.Duration
		WithDebugLog     bool
	}
)

func NewGRPCConfig(cfg *config.Config) Config {
	return Config{
		ListenAddr:       cfg.GRPC.ListenAddr,
		KeepaliveTime:    cfg.GRPC.KeepaliveTime,
		KeepaliveTimeout: cfg.GRPC.KeepaliveTimeout,
		WithReflection:   cfg.GRPC.RegisterReflectionServer,
		WithDebugLog:     cfg.DebugLog,
	}
}
