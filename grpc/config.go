package grpc

import (
	"strings"
	"time"

	"github.com/nenormalka/bishamon"
	lilith "github.com/nenormalka/lilith/methods"
	"github.com/nenormalka/lilith/patterns"

	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/types"
)

type (
	Config struct {
		Throttler             Throttler
		ListenAddr            string
		WithReflection        bool
		KeepaliveTime         time.Duration
		KeepaliveTimeout      time.Duration
		WithDebugLog          bool
		WithServerMetrics     bool
		WithMetadata          bool
		LogRedactor           *bishamon.Redactor
		MaxReceiveMessageSize int
		MaxSendMessageSize    int
	}

	Throttler interface {
		Accept(method string) bool
	}

	LoggerThrottler struct {
		generators map[string]*patterns.BoolGenerator
	}

	MockThrottler struct{}
)

func NewGRPCConfig(cfg *config.Config) *Config {
	return &Config{
		Throttler:             getThrottler(cfg),
		ListenAddr:            types.CheckAddr(cfg.GRPC.ListenAddr),
		KeepaliveTime:         cfg.GRPC.KeepaliveTime,
		KeepaliveTimeout:      cfg.GRPC.KeepaliveTimeout,
		WithReflection:        cfg.GRPC.RegisterReflectionServer,
		WithDebugLog:          cfg.DebugLog,
		WithMetadata:          cfg.GRPC.WithMetadata,
		WithServerMetrics:     cfg.EnableServerMetrics,
		MaxReceiveMessageSize: cfg.GRPC.MaxReceiveMessageSize,
		MaxSendMessageSize:    cfg.GRPC.MaxReceiveMessageSize,
	}
}

func getThrottler(cfg *config.Config) Throttler {
	methods := lilith.ArrayToMapValues[[]string, string](strings.Split(cfg.ThrottleConfig.Methods, ","))

	if !cfg.ThrottleConfig.EnableThrottle || len(methods) == 0 {
		return &MockThrottler{}
	}

	generators := make(map[string]*patterns.BoolGenerator, len(methods))
	for method := range methods {
		generators[method] = patterns.NewBoolGenerator(patterns.ChanceQuarter)
	}

	return &LoggerThrottler{
		generators: generators,
	}
}

func (*MockThrottler) Accept(_ string) bool {
	return true
}

func (lt *LoggerThrottler) Accept(method string) bool {
	gen, ok := lt.generators[method]
	if !ok {
		return true
	}

	return gen.Bool()
}
