package types

import (
	"database/sql"
	"time"

	"github.com/jackc/pgx/v4"

	lilith "github.com/nenormalka/lilith/methods"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DBMetrics = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "connections",
			Subsystem: "sql",
			Name:      "call_duration_seconds",
			Help:      "query duration seconds",
			Buckets:   []float64{.005, .01, .025, .05, .075, .1, .15, .2, .25, .5, 1, 2.5},
		}, []string{"query", "service", "error"},
	)

	DBErrorMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "connections",
		Subsystem: "sql",
		Name:      "error_total",
		Help:      "Number of db errors.",
	}, []string{"query_name"})

	GRPCErrorMetrics = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "error_total",
		Help:      "Number of grpc errors.",
	})

	GRPCPanicMetrics = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "panic_total",
		Help:      "Number of grpc panic.",
	})

	KafkaConsumerGroup = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kafka",
			Subsystem: "consumer_group",
			Name:      "duration_consume",
			Help:      "Consumer group consume duration",
		},
		[]string{"consumer_group", "topic", "error"},
	)
)

func GRPCPanicInc() {
	GRPCPanicMetrics.Inc()
}

func GRPCErrorInc() {
	GRPCErrorMetrics.Inc()
}

func WithKafkaConsumerGroupMetrics(groupName, topic string, err error, duration float64) {
	KafkaConsumerGroup.
		WithLabelValues(groupName, topic, lilith.Ternary(isError(err), "true", "false")).
		Observe(duration)
}

func WithSQLMetrics(
	queryName, serviceName string,
	callFunc func() error,
) error {
	var err error
	defer func(start time.Time) {
		DBMetrics.
			WithLabelValues(queryName, serviceName, lilith.Ternary(isError(err), "true", "false")).
			Observe(time.Since(start).Seconds())
	}(time.Now())

	err = callFunc()
	if isError(err) {
		DBErrorMetrics.
			With(prometheus.Labels{
				"query_name": queryName,
			}).
			Inc()
	}

	return err
}

func isError(err error) bool {
	switch err {
	case sql.ErrNoRows, pgx.ErrNoRows, nil:
		return false
	default:
		return true
	}
}
