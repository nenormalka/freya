package types

import (
	"context"

	"go.uber.org/zap"
)

type (
	ServiceList []Runnable

	ServicePool struct {
		p      []Runnable
		logger *zap.Logger
	}
)

func NewServicePool(sl ServiceList, logger *zap.Logger) *ServicePool {
	return &ServicePool{
		p:      sl,
		logger: logger,
	}
}

func (p *ServicePool) Start(ctx context.Context) error {
	return start(ctx, p.p, p.logger, runnableService)
}

func (p *ServicePool) Stop(ctx context.Context) {
	stop(ctx, p.p, p.logger, runnableService)
}
