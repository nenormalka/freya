package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nenormalka/freya/config"
)

func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	var level zapcore.Level

	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("level  UnmarshalText %w", err)
	}

	configZap := zap.NewProductionConfig()
	configZap.Level = zap.NewAtomicLevelAt(level)
	configZap.InitialFields = map[string]any{
		"service":    cfg.AppName,
		"release.id": cfg.ReleaseID,
	}

	logger, err := configZap.Build()
	if err != nil {
		return nil, fmt.Errorf("zap  biuld %w", err)
	}

	return logger, nil
}
