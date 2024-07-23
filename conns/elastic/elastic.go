package elastic

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nenormalka/freya/types"

	estransport "github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.elastic.co/apm/module/apmelasticsearch/v2"
	"go.uber.org/zap"
)

const (
	// maxBytesPerBulkRequest 2MB
	maxBytesPerBulkRequest = 2000000
	bulkNumWorkers         = 10
)

type (
	ElasticConn struct {
		client *elasticsearch.Client
		logger *zap.Logger
	}

	BulkIndexerOpts func(cfg *esutil.BulkIndexerConfig)

	BulkIndexer interface {
		Stats() esutil.BulkIndexerStats
		Add(ctx context.Context, item esutil.BulkIndexerItem) error
	}
)

func WithBulkIndexerMaxBytesPerRequest(maxBytesPerRequest int) BulkIndexerOpts {
	return func(cfg *esutil.BulkIndexerConfig) {
		cfg.FlushBytes = maxBytesPerRequest
	}
}

func WithBulkIndexerNumWorkers(numWorkers int) BulkIndexerOpts {
	return func(cfg *esutil.BulkIndexerConfig) {
		cfg.NumWorkers = numWorkers
	}
}

func WithCustomBulkIndexerConfig(customCfg *esutil.BulkIndexerConfig) BulkIndexerOpts {
	return func(cfg *esutil.BulkIndexerConfig) {
		*cfg = *customCfg
	}
}

func DefaultBulkIndexerFailureFunc(logger *zap.Logger, indexName string) func(
	ctx context.Context,
	item esutil.BulkIndexerItem,
	res esutil.BulkIndexerResponseItem,
	err error,
) {
	return func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
		if err != nil {
			logger.Error(fmt.Sprintf("error indexing item for index: %s", indexName), zap.Error(err))

			return
		}

		logger.Error(
			fmt.Sprintf("error indexing index: %s type: %s reason: %s cause type: %s cause reason: %s",
				indexName, res.Error.Type, res.Error.Reason, res.Error.Cause.Type, res.Error.Cause.Reason))
	}
}

func NewElastic(cfg Config) (*elasticsearch.Client, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:         strings.Split(cfg.DSN, ","),
		RetryOnStatus:     []int{502, 503, 504, 429},
		RetryBackoff:      func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
		Transport:         apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
		MaxRetries:        cfg.MaxRetries,
		EnableDebugLogger: cfg.WithLogger,
		Logger: func() estransport.Logger {
			if !cfg.WithLogger {
				return nil
			}

			return &estransport.JSONLogger{
				Output:             os.Stdout,
				EnableRequestBody:  true,
				EnableResponseBody: true,
			}
		}(),
	})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch.NewClient: %w", err)
	}

	res, err := es.Ping()
	if err != nil {
		return nil, fmt.Errorf("elasticseach.Ping: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("elasticseach.Ping: %s", res.String())
	}

	return es, nil
}

func NewElasticConn(
	client *elasticsearch.Client,
	logger *zap.Logger,
) *ElasticConn {
	if client == nil {
		return nil
	}

	return &ElasticConn{
		client: client,
		logger: logger,
	}
}

func (ec *ElasticConn) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, client *elasticsearch.Client) error,
) error {
	return types.WithElasticMetrics(queryName, func() error {
		return callFunc(ctx, ec.client)
	})
}

func (ec *ElasticConn) CallContextBulkIndexer(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, bi BulkIndexer) error,
	opts ...BulkIndexerOpts,
) error {
	cfg := esutil.BulkIndexerConfig{
		FlushBytes: maxBytesPerBulkRequest,
		NumWorkers: bulkNumWorkers,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.Client = ec.client

	bulkIndexer, err := esutil.NewBulkIndexer(cfg)
	if err != nil {
		return fmt.Errorf("esutil.NewBulkIndexer: %w", err)
	}

	startAt := time.Now()
	defer func() {
		if err = bulkIndexer.Close(ctx); err != nil {
			ec.logger.Error("bulkIndexer.Close", zap.Error(err))
		}

		stats := bulkIndexer.Stats()
		if stats.NumFailed == 0 {
			ec.logger.Info(fmt.Sprintf(
				"successfully flushed: %d added: %d created: %d updated: %d indexed: %d %s in %d seconds",
				stats.NumFlushed,
				stats.NumAdded,
				stats.NumCreated,
				stats.NumUpdated,
				stats.NumIndexed,
				queryName,
				int(time.Since(startAt).Seconds()),
			))

			return
		}

		ec.logger.Error(fmt.Sprintf(
			"indexed [%d] %s with [%d] errors in %d seconds",
			stats.NumFlushed, queryName, stats.NumFailed, int(time.Since(startAt).Seconds()),
		))
	}()

	return types.WithElasticMetrics(queryName, func() error {
		return callFunc(ctx, bulkIndexer)
	})
}
