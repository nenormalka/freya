package types

import (
	"database/sql"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
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

	GRPCPanicMetrics = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "panic_total",
		Help:      "Number of grpc panic.",
	})

	KafkaConsumerGroupMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kafka",
			Subsystem: "consumer_group",
			Name:      "duration_consume",
			Help:      "Consumer group consume duration",
		},
		[]string{"consumer_group", "topic", "error"},
	)

	KafkaSyncProducerMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kafka",
			Subsystem: "sync_producer",
			Name:      "produce_count",
			Help:      "Sync producer produce count",
		},
		[]string{"topic", "error"},
	)

	GaugeAppState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "application",
			Subsystem: "app",
			Name:      "state",
			Help:      "Version and env of application",
		}, []string{"version", "start_date"},
	)

	ServerGRPCMetrics = grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(),
	)
)

func SetApplicationMetrics(version string) {
	GaugeAppState.WithLabelValues(version, time.Now().String()).Set(1)
}

func GRPCPanicInc() {
	GRPCPanicMetrics.Inc()
}

func KafkaSyncProducerMetricsF(topic string, err error) {
	KafkaSyncProducerMetrics.
		WithLabelValues(topic, errToBoolString(err)).
		Inc()
}

func KafkaConsumerGroupMetricsF(groupName, topic string, err error, duration float64) {
	KafkaConsumerGroupMetrics.
		WithLabelValues(groupName, topic, errToBoolString(err)).
		Observe(duration)
}

func WithSQLMetrics(
	queryName, serviceName string,
	callFunc func() error,
) error {
	var err error
	defer func(start time.Time) {
		DBMetrics.
			WithLabelValues(queryName, serviceName, errToBoolString(err)).
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

func errToBoolString(err error) string {
	return lilith.Ternary(isError(err), "true", "false")
}

func isError(err error) bool {
	switch err {
	case sql.ErrNoRows, pgx.ErrNoRows, nil:
		return false
	default:
		return true
	}
}
