package postgrelock

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	lilith "github.com/nenormalka/lilith/patterns"

	"github.com/nenormalka/freya/conns/connectors"
)

const (
	retriesDefault = 3
	delayDefault   = 2 * time.Second
)

const (
	lockTxSQL = "SELECT pg_try_advisory_xact_lock(%d);"
	lockSQL   = "SELECT pg_try_advisory_lock(%d);"
	unlockSQL = "SELECT pg_advisory_unlock(%d);"
)

var (
	ErrDontDo   = errors.New("don't do")
	ErrNotAllow = errors.New("not allow")
	ErrEmptyKey = errors.New("empty key")
)

type (
	CommonLock[T connectors.ConnectDB, M connectors.ConnectTx] struct {
		db  connectors.DBConnector[T, M]
		cfg *LockConfig
	}

	LockConfig struct {
		retries int
		delay   time.Duration
	}

	LockFunc[T connectors.ConnectDB]   func(ctx context.Context, lockQuery, unlock string, db T) error
	LockTxFunc[T connectors.ConnectTx] func(ctx context.Context, lockQuery string, db T) error

	LockOption func(lc *LockConfig)
)

func WithRetriesOption(retries int) LockOption {
	return func(lc *LockConfig) {
		lc.retries = retries
	}
}

func WithDelayOption(delay time.Duration) LockOption {
	return func(lc *LockConfig) {
		lc.delay = delay
	}
}

func NewCommonLock[T connectors.ConnectDB, M connectors.ConnectTx](
	db connectors.DBConnector[T, M],
	opts ...LockOption,
) CommonLock[T, M] {
	cl := CommonLock[T, M]{
		db: db,
	}

	cfg := &LockConfig{
		retries: retriesDefault,
		delay:   delayDefault,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cl.cfg = cfg

	return cl
}

func (cl *CommonLock[T, M]) DoUnderLock(ctx context.Context, key string, f LockFunc[T]) error {
	if key == "" {
		return ErrEmptyKey
	}

	if err := cl.process(ctx, func(ctx context.Context) error {
		if err := cl.db.CallContext(ctx, "DoUnderLock", func(ctx context.Context, db T) error {
			keyDB := cl.getKey(key)

			if err := f(ctx, cl.getLockQuery(keyDB), cl.getUnlockQuery(keyDB), db); err != nil {
				return fmt.Errorf("DoUnderLock f; %w", err)
			}

			return nil
		}); err != nil {
			return fmt.Errorf("db.CallContext; %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("process; %w", err)
	}

	return nil
}

func (cl *CommonLock[T, M]) DoUnderLockTx(ctx context.Context, key string, f LockTxFunc[M]) error {
	if key == "" {
		return ErrEmptyKey
	}

	if err := cl.process(ctx, func(ctx context.Context) error {
		if err := cl.db.CallTransaction(ctx, "DoUnderLockTx", func(ctx context.Context, db M) error {
			if err := f(ctx, cl.getLockTxQuery(cl.getKey(key)), db); err != nil {
				return fmt.Errorf("DoUnderLockTx f; %w", err)
			}

			return nil
		}); err != nil {
			return fmt.Errorf("db.CallTransaction; %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("process; %w", err)
	}

	return nil
}

func (cl *CommonLock[T, M]) process(
	ctx context.Context,
	f func(ctx context.Context) error,
) error {
	e := lilith.Retry(func(ctx context.Context) (bool, error) {
		if err := f(ctx); err != nil {
			return false, fmt.Errorf("retry f err: %w", err)
		}

		return true, nil
	}, cl.cfg.retries, cl.cfg.delay)

	done, err := e(ctx)
	if err != nil {
		return fmt.Errorf("retry err: %w", err)
	}

	if !done {
		return ErrDontDo
	}

	return nil
}

func (cl *CommonLock[T, M]) getLockTxQuery(key int64) string {
	return cl.getQuery(lockTxSQL, key)
}

func (cl *CommonLock[T, M]) getLockQuery(key int64) string {
	return cl.getQuery(lockSQL, key)
}

func (cl *CommonLock[T, M]) getUnlockQuery(key int64) string {
	return cl.getQuery(unlockSQL, key)
}

func (cl *CommonLock[T, M]) getQuery(query string, key int64) string {
	return fmt.Sprintf(query, key)
}

func (cl *CommonLock[T, M]) getKey(key string) int64 {
	h := fnv.New64a()
	h.Write([]byte(key))

	return int64(h.Sum64())
}
