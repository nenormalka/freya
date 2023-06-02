package elastic

import (
	"fmt"
	"os"
	"strings"
	"time"

	estransport "github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
)

func NewElastic(cfg Config) (*elasticsearch.Client, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:         strings.Split(cfg.DSN, ","),
		RetryOnStatus:     []int{502, 503, 504, 429},
		RetryBackoff:      func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
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
