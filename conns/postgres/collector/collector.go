package collector

import (
	"errors"
	"fmt"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
)

func CollectDBStats(dbName string, getter sqlstats.StatsGetter) error {
	collector := sqlstats.NewStatsCollector(dbName, getter)
	if err := prometheus.Register(collector); err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		return fmt.Errorf("register pgx sqlstats err: %w", err)
	}

	return nil
}
