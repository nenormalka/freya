package redlock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/redis/types"
)

const (
	expireDefault     = 10 * time.Second
	retryDelayDefault = 2 * time.Second
	retryCountDefault = 3
)

type (
	RedLock struct {
		redis      connectors.DBConnector[*redis.Client, *types.RedisTx]
		retryCount int
		retryDelay time.Duration
		expires    time.Duration
	}

	RedLockOption func(rl *RedLock)
)

func WithRetryCountOption(retryCount int) RedLockOption {
	return func(cl *RedLock) {
		cl.retryCount = retryCount
	}
}

func WithRetryDelayOption(delay time.Duration) RedLockOption {
	return func(cl *RedLock) {
		cl.retryDelay = delay
	}
}

func WithExpireOption(expire time.Duration) RedLockOption {
	return func(cl *RedLock) {
		cl.expires = expire
	}
}

func NewRedLock(
	redis connectors.DBConnector[*redis.Client, *types.RedisTx],
	opts ...RedLockOption,
) (*RedLock, error) {
	rl := &RedLock{
		redis:      redis,
		retryCount: retryCountDefault,
		retryDelay: retryDelayDefault,
		expires:    expireDefault,
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl, nil
}

func (rl *RedLock) DoUnderLock(ctx context.Context, key string, f func(ctx context.Context) error) error {
	if err := rl.redis.CallContext(ctx, "DoUnderLock", func(ctx context.Context, client *redis.Client) error {
		rs := redsync.New(goredis.NewPool(client))
		mutex := rs.NewMutex(
			key,
			redsync.WithTries(rl.retryCount),
			redsync.WithRetryDelay(rl.retryDelay),
			redsync.WithExpiry(rl.expires),
		)

		var err error

		if err = mutex.LockContext(ctx); err != nil {
			return fmt.Errorf("failed to lock: %w", err)
		}

		defer func() {
			if _, errUC := mutex.UnlockContext(ctx); errUC != nil {
				err = errors.Join(errUC, err)
			}
		}()

		return f(ctx)
	}); err != nil {
		return fmt.Errorf("failed to do under lock: %w", err)
	}

	return nil
}
