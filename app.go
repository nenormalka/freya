package freya

import (
	"context"
	"fmt"
	"time"

	"github.com/nenormalka/freya/types"

	"go.uber.org/zap"
)

const (
	shutdownTimeout = 10 * time.Second
)

type (
	App struct {
		servers  *types.ServerPool
		services *types.ServicePool

		logger *zap.Logger
	}
)

func NewApp(
	servers *types.ServerPool,
	services *types.ServicePool,
	logger *zap.Logger,
) *App {
	return &App{
		servers:  servers,
		services: services,
		logger:   logger,
	}
}

func (c *App) Run(ctx context.Context) error {
	c.logger.Info("Services start")

	if err := c.services.Start(ctx); err != nil {
		return fmt.Errorf("services start err: %w", err)
	}

	c.logger.Info("Servers start")
	if err := c.servers.Start(ctx); err != nil {
		return fmt.Errorf("servers start err: %w", err)
	}

	c.logger.Info("Application is ready 🐣")

	<-ctx.Done()

	sdCtx, sdCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer sdCancel()

	c.logger.Info("Stopping servers...")
	c.servers.Stop(sdCtx)

	c.logger.Info("Stopping services...")
	c.services.Stop(sdCtx)

	c.logger.Info("Gracefully stopped, bye bye 👋")

	return nil
}
