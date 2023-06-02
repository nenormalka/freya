package types

import (
	"database/sql"
	"time"

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
)

func GRPCPanicInc() {
	GRPCPanicMetrics.Inc()
}

func GRPCErrorInc() {
	GRPCErrorMetrics.Inc()
}

func WithSQLMetrics(
	queryName, serviceName string,
	callFunc func() error,
) error {
	var err error
	defer func(start time.Time) {
		DBMetrics.
			WithLabelValues(queryName, serviceName, errLabel(err)).
			Observe(time.Since(start).Seconds())
	}(time.Now())

	err = callFunc()
	return err
}

func errLabel(err error) string {
	if err == nil {
		return "false"
	}
	switch err {
	case sql.ErrNoRows:
		return "false"
	default:
		return "true"
	}
}
