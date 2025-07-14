package postrgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/postgres/collector"

	_ "go.elastic.co/apm/module/apmsql/v2/pgxv4"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"go.elastic.co/apm/module/apmsql/v2"
)

func NewSQLX(config PostgresConfig) (map[string]*sqlx.DB, error) {
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

func getMSSQLXConnectFromDSN(dsn string) (*sql.DB, error) {
	parts := strings.Split(dsn, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dsn parts step 1: %s", dsn)
	}

	if parts[0] != "sqlserver" {
		return nil, fmt.Errorf("invalid scheme: %s", dsn)
	}

	scheme := parts[0]

	parts = strings.Split(parts[1], "?")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dsn parts step 2: %s", dsn)
	}

	database := parts[1]

	parts = strings.Split(parts[0], "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dsn parts step 3: %s", dsn)
	}

	host := parts[1]

	parts = strings.Split(parts[0], ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dsn parts step 4: %s", dsn)
	}

	u := url.URL{
		Scheme:   scheme,
		User:     url.UserPassword(parts[0], parts[1]),
		Host:     host,
		RawQuery: database,
	}

	return sql.Open("mssql", u.String())
}

func newDb(cfg DBConfig) (*sqlx.DB, error) {
	var (
		db  *sql.DB
		err error
	)

	if cfg.DBType == connectors.MsSQLDBType {
		db, err = getMSSQLXConnectFromDSN(cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("get db mssql err: %w", err)
		}
	} else {
		db, err = apmsql.Open("pgx", cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("open apmsql err: %w", err)
		}
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

	return sqlx.NewDb(db, cfg.DBType), nil
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
