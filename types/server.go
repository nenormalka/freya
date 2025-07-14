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

func CheckAddr(addr string) string {
	if addr == "" || strings.Contains(addr, ":") {
		return addr
	}

	return ":" + addr
}

func StartServerWithWaiting(
	ctx context.Context,
	logger *zap.Logger,
	f func(errCh chan error),
) error {
	errCh := make(chan error)
	ctxT, cancel := context.WithTimeout(ctx, waitingTime)
	defer cancel()
	defer func() {
		go func() {
			for err := range errCh {
				if err != nil {
					logger.Error("server err", zap.Error(err))
				}
			}
		}()
	}()

	go f(errCh)

	select {
	case <-ctxT.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
