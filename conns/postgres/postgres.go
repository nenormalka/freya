package postrgres

import (
	"errors"
	"fmt"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/pgxv4"
)

const (
	driverName = "pgx"
)

func NewPostgres(config PostgresConfig) (*sqlx.DB, error) {
	if config.DSN == "" {
		return nil, nil
	}

	db, err := apmsql.Open(driverName, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("open apmsql err: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}

		if errClose := db.Close(); errClose != nil {
			err = fmt.Errorf("%w: %s", err, errClose.Error())
		}
	}()

	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	cfg, err := pgx.ParseConfig(config.DSN)
	if err != nil {
		return nil, err
	}

	collector := sqlstats.NewStatsCollector(cfg.Database, db)
	if err = prometheus.Register(collector); err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db err: %w", err)
	}

	return sqlx.NewDb(db, driverName), nil
}
