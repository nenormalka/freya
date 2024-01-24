package couchlock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/couchbase/gocb/v2"
	lilith "github.com/nenormalka/lilith/patterns"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/couchbase/logger"
	"github.com/nenormalka/freya/conns/couchbase/types"
)

const (
	ttlDefault = 10 * time.Second
	retryCount = 5
)

var (
	errWrongCount = errors.New("wrong count")
	errDontDo     = errors.New("don't do")
)

type (
	CouchLock struct {
		collection connectors.DBConnector[*gocb.Collection, *types.CollectionTx]
		retryCount int
		ttl        time.Duration
		logger     Logger
	}

	Logger interface {
		Info(msg string, keysAndValues ...any)
	}

	CouchLockOption func(cl *CouchLock)
)

func WithRetryCountOption(retryCount int) CouchLockOption {
	return func(cl *CouchLock) {
		cl.retryCount = retryCount
	}
}

func WithTTLOption(ttl time.Duration) CouchLockOption {
	return func(cl *CouchLock) {
		cl.ttl = ttl
	}
}

func WithZapLoggerOption(l *zap.Logger) CouchLockOption {
	return func(cl *CouchLock) {
		cl.logger = logger.NewZapLogger(l.With(zap.String("target", "couchlock")))
	}
}

// Чтобы лок работал правильно, в конфиге должен быть указан ТОЛЬКО ОДИН кластер
// https://docs.couchbase.com/go-sdk/current/howtos/kv-operations.html#atomicity-across-data-centers
func NewCouchLock(
	c *conns.Conns,
	bucketName, collectionName string,
	opts ...CouchLockOption,
) (*CouchLock, error) {
	cb, err := c.GetCouchbase()
	if err != nil {
		return nil, fmt.Errorf("get couchbase err: %w", err)
	}

	coll, err := cb.GetCollection(bucketName, collectionName)
	if err != nil {
		return nil, fmt.Errorf("get collection err: %w", err)
	}

	cl := &CouchLock{
		collection: coll,
		retryCount: retryCount,
		ttl:        ttlDefault,
		logger:     logger.NewDefaultLogger(),
	}

	for _, opt := range opts {
		opt(cl)
	}

	return cl, nil
}

func (c *CouchLock) DoUnderLock(
	ctx context.Context,
	key string,
	f func(ctx context.Context) error,
) error {
	r := rand.Intn(10)

	e := lilith.Retry(func(ctx context.Context) (bool, error) {
		if errC := c.collection.CallContext(ctx, key, func(ctx context.Context, col *gocb.Collection) error {
			count, err := col.Binary().Increment(key, &gocb.IncrementOptions{
				Context: ctx,
				Initial: 1,
				Delta:   1,
				Expiry:  c.ttl,
			})
			if err != nil {
				c.logger.Info("collection.Binary().Increment", "err", err)

				return fmt.Errorf("collection.Binary().Increment; %w", err)
			}

			defer func() {
				if _, err = col.Binary().Decrement(key, &gocb.DecrementOptions{
					Context: ctx,
					Delta:   1,
				}); err != nil {
					c.logger.Info("collection.Binary().Decrement", "err", err)
				}
			}()

			if count.Content() > 1 {
				c.logger.Info("count.Content() > 1", "count", int64(count.Content()))

				return errWrongCount
			}

			if err = f(ctx); err != nil {
				c.logger.Info("function under lock", "err", err)

				return fmt.Errorf("function under lock %w", err)
			}

			return nil
		}); errC != nil {
			return false, fmt.Errorf("collection.CallContext; %w", errC)
		}

		return true, nil
	}, c.retryCount, time.Duration(r)*time.Second)

	done, err := e(ctx)
	if err != nil {
		return fmt.Errorf("retry err: %w", err)
	}

	if !done {
		return errDontDo
	}

	return nil
}
