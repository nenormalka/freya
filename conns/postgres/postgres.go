package postrgres

import (
	"fmt"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/postgres/collector"

	"github.com/jmoiron/sqlx"
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

	m := make(map[string]*sqlx.DB)

	for i := range config.Configs {
		if config.Configs[i].Type != connectors.SqlxConnType {
			continue
		}

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

	if err = collector.CollectDBStats(cfg.Name, db); err != nil {
		return nil, fmt.Errorf("failed to collect db stats: %w", err)
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
