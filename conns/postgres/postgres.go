package postrgres

import (
	"errors"
	"fmt"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/pgxv4"
)

const (
	driverName = "pgx"
)

func NewPostgres(config PostgresConfig) (map[string]*sqlx.DB, error) {
	if len(config.Configs) == 0 {
		return nil, nil
	}

	m := make(map[string]*sqlx.DB, len(config.Configs))

	for i := range config.Configs {
		db, err := newDb(config.Configs[i])
		if err != nil {
			return nil, fmt.Errorf("new db err: %w", err)
		}

		m[config.Configs[i].Name] = db
	}

	return m, nil
}

func newDb(cfg DBConfig) (*sqlx.DB, error) {
	db, err := apmsql.Open(driverName, cfg.DSN)
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

	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	collector := sqlstats.NewStatsCollector(cfg.Name, db)
	if err = prometheus.Register(collector); err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db err: %w", err)
	}

	return sqlx.NewDb(db, driverName), nil
}

func newConns[T any](poolDB map[string]*sqlx.DB, f func(nameConn string) T) map[string]T {
	if len(poolDB) == 0 {
		return nil
	}

	m := make(map[string]T, len(poolDB))

	for i := range poolDB {
		m[i] = f(i)
	}

	return m
}
