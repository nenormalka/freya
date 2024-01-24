package couchbase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/couchbase/logger"
	txtype "github.com/nenormalka/freya/conns/couchbase/types"
	"github.com/nenormalka/freya/types"
)

const (
	defaultTimeout = 5 * time.Second
)

var (
	ErrEmptyBuckets = errors.New("empty buckets")
)

type (
	Couchbase struct {
		buckets map[string]*gocb.Bucket
		cluster *gocb.Cluster
		appName string
	}

	Collection struct {
		collection     *gocb.Collection
		cluster        *gocb.Cluster
		bucket         *gocb.Bucket
		collectionName string
		bucketName     string
		appName        string
	}
)

func NewCouchbase(l *zap.Logger, cfg Config) (*Couchbase, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	if len(cfg.Buckets) == 0 {
		return nil, ErrEmptyBuckets
	}

	cluster, err := gocb.Connect(cfg.DSN, gocb.ClusterOptions{
		Username: cfg.User,
		Password: cfg.Password,
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout: defaultTimeout,
			QueryTimeout:   defaultTimeout,
			SearchTimeout:  defaultTimeout,
		},
	})

	if cfg.EnableDebug {
		gocb.SetLogger(logger.NewZapLogger(l))
	}

	if err != nil {
		return nil, fmt.Errorf("connect to couchbase: %w", err)
	}

	if _, err = cluster.Ping(&gocb.PingOptions{Timeout: defaultTimeout}); err != nil {
		return nil, fmt.Errorf("ping couchbase: %w", err)
	}

	buckets := make(map[string]*gocb.Bucket, len(cfg.Buckets))

	for _, bucketName := range cfg.Buckets {
		bucket := cluster.Bucket(bucketName)
		if err = bucket.WaitUntilReady(defaultTimeout, nil); err != nil {
			return nil, fmt.Errorf("couchbase wait until ready bucket %s: %w", bucketName, err)
		}

		buckets[bucketName] = bucket
	}

	return &Couchbase{
		buckets: buckets,
		cluster: cluster,
		appName: cfg.AppName,
	}, nil
}

func (c *Couchbase) GetCollection(bucketName, collectionName string) (connectors.DBConnector[*gocb.Collection, *txtype.CollectionTx], error) {
	bucket, ok := c.buckets[bucketName]
	if !ok {
		return nil, fmt.Errorf("bucket %s not found", bucketName)
	}

	if err := bucket.Collections().CreateCollection(
		gocb.CollectionSpec{
			Name:      collectionName,
			ScopeName: bucket.DefaultScope().Name(),
		},
		&gocb.CreateCollectionOptions{
			Timeout: defaultTimeout,
		},
	); err != nil && !errors.Is(err, gocb.ErrCollectionExists) {
		return nil, fmt.Errorf("creating collection in couchbase: %w", err)
	}

	return &Collection{
		collection:     bucket.Collection(collectionName),
		cluster:        c.cluster,
		bucketName:     bucketName,
		collectionName: collectionName,
		appName:        c.appName,
		bucket:         bucket,
	}, nil
}

func (c *Collection) CallContext(
	ctx context.Context,
	query string,
	f func(ctx context.Context, collection *gocb.Collection) error,
) error {
	return types.WithCouchbaseMetrics(
		c.bucketName,
		c.collectionName,
		query,
		c.appName,
		func() error {
			return f(ctx, c.collection)
		})
}

func (c *Collection) CallTransaction(
	ctx context.Context,
	query string,
	f func(ctx context.Context, c *txtype.CollectionTx) error,
) error {
	return types.WithCouchbaseMetrics(
		c.bucketName,
		c.collectionName,
		query,
		c.appName,
		func() error {
			if _, err := c.cluster.Transactions().Run(func(ctxTx *gocb.TransactionAttemptContext) error {
				return f(
					ctx,
					&txtype.CollectionTx{
						Ctx:        ctxTx,
						Collection: c.collection,
						Bucket:     c.bucket,
					})
			}, nil); err != nil {
				return fmt.Errorf("couchbase transaction: %w", err)
			}

			return nil
		})
}
