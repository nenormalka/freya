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
	"go.elastic.co/apm/module/apmelasticsearch/v2"
)

type (
	ElasticConn struct {
		client *elasticsearch.Client
	}
)

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

func NewElasticConn(client *elasticsearch.Client) *ElasticConn {
	if client == nil {
		return nil
	}

	return &ElasticConn{
		client: client,
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
