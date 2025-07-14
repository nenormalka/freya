package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	retryDefault = 3
)

var (
	ErrEmptyKeys = errors.New("empty keys")
	ErrMaxRetry  = errors.New("max retry")
)

type (
	RedisTx struct {
		retry int
		f     func(ctx context.Context, fn func(tx *redis.Tx) error, keys ...string) error
	}
)

func NewRedisTx(f func(ctx context.Context, fn func(tx *redis.Tx) error, keys ...string) error) *RedisTx {
	return &RedisTx{
		retry: retryDefault,
		f:     f,
	}
}

func (r *RedisTx) SetRetry(count int) {
	r.retry = count
}

func (r *RedisTx) Do(
	ctx context.Context,
	keys []string,
	fn func(tx *redis.Tx) error,
) error {
	if len(keys) == 0 {
		return ErrEmptyKeys
	}

	for i := 0; i < r.retry; i++ {
		err := r.f(ctx, fn, keys...)
		if err == nil {
			return nil
		}

		if errors.Is(err, redis.TxFailedErr) {
			continue
		}

		return fmt.Errorf("failed to execute transaction: %v", err)
	}

	return ErrMaxRetry
}
