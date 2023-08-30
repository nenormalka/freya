package types

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	waitingTime = 300 * time.Millisecond
)

type (
	ServerList []Runnable

	ServerPool struct {
		p      []Runnable
		logger *zap.Logger
	}
)

func NewServerPool(sl ServerList, logger *zap.Logger) *ServerPool {
	return &ServerPool{
		p:      sl,
		logger: logger,
	}
}

func (p *ServerPool) Start(ctx context.Context) error {
	return start(ctx, p.p, p.logger, runnableServer)
}

func (p *ServerPool) Stop(ctx context.Context) {
	stop(ctx, p.p, p.logger, runnableServer)
}

func StartServerWithWaiting(ctx context.Context, f func(errCh chan error)) error {
	errCh := make(chan error)
	ctxT, cancel := context.WithTimeout(ctx, waitingTime)
	defer cancel()
	defer close(errCh)

	go f(errCh)

	select {
	case <-ctxT.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func CheckAddr(addr string) string {
	if addr == "" || strings.Contains(addr, ":") {
		return addr
	}

	return ":" + addr
}
