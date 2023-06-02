package sentry

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
)

const (
	sentryFlushTimeout = 2 * time.Second
)

func NewSentry(cfg Config) (*sentry.Hub, error) {
	if cfg.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:         cfg.DSN,
			Environment: cfg.Environment,
			Release:     cfg.ReleaseID,
		}); err != nil {
			return nil, fmt.Errorf("sentry.Init %w", err)
		}
	}

	sentry.Flush(sentryFlushTimeout)

	return sentry.CurrentHub(), nil
}
