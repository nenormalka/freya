package grpc

import (
	"time"

	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/types"
)

type (
	Config struct {
		ListenAddr        string
		WithReflection    bool
		KeepaliveTime     time.Duration
		KeepaliveTimeout  time.Duration
		WithDebugLog      bool
		WithServerMetrics bool
	}
)

func NewGRPCConfig(cfg *config.Config) Config {
	return Config{
		ListenAddr:        types.CheckAddr(cfg.GRPC.ListenAddr),
		KeepaliveTime:     cfg.GRPC.KeepaliveTime,
		KeepaliveTimeout:  cfg.GRPC.KeepaliveTimeout,
		WithReflection:    cfg.GRPC.RegisterReflectionServer,
		WithDebugLog:      cfg.DebugLog,
		WithServerMetrics: cfg.EnableServerMetrics,
	}
}
