package postrgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/nenormalka/freya/conns/connectors"
	dbtypes "github.com/nenormalka/freya/conns/postgres/types"
	"github.com/nenormalka/freya/types"

	"github.com/nenormalka/freya/config"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.elastic.co/apm/module/apmpgx/v2"
	"go.uber.org/zap"
)

type (
	PGXPoolConn struct {
		*pgxpool.Pool
		logger  *zap.Logger
		appName string
	}
)

func NewPGXPool(
	ctx context.Context,
	config PostgresConfig,
	logger *zap.Logger,
	cfg *config.Config,
) (map[string]connectors.DBConnector[dbtypes.PgxConn, *pgxpool.Tx], error) {
	if len(config.Configs) == 0 {
		return nil, nil
	}

	poolDB := make(map[string]*PGXPoolConn)

	for i := range config.Configs {
		if config.Configs[i].Type != connectors.PgxConnType {
			continue
		}

		pool, err := newPgxPoolConn(ctx, config.Configs[i], logger, cfg.AppName)
		if err != nil {
			return nil, fmt.Errorf("new pgx pool err: %w", err)
		}

		poolDB[config.Configs[i].Name] = pool
	}

	return nil, nil
}

func (c *PGXPoolConn) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, db dbtypes.PgxConn) error,
) error {
	return types.WithSQLMetrics(queryName, c.appName, func() error {
		return callFunc(ctx, dbtypes.PgxConn{PgxQuerier: c})
	})
}

func (c *PGXPoolConn) CallTransaction(
	ctx context.Context,
	txName string,
	callFunc func(ctx context.Context, tx pgxtype.Querier) error,
) error {
	return types.WithSQLMetrics(txName, c.appName, func() error {
		tx, err := c.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}

		if err = callFunc(ctx, tx); err != nil {
			if rErr := tx.Rollback(ctx); rErr != nil {
				c.logger.Error(fmt.Sprintf("failed rollback transaction: %s", txName), zap.Error(rErr))
			}

			return err
		}

		return tx.Commit(ctx)
	})
}

func (c *PGXPoolConn) Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	return pgxscan.Select(ctx, c, dst, query, args...)
}

func (c *PGXPoolConn) Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	return pgxscan.Get(ctx, c, dst, query, args...)
}

func (c *PGXPoolConn) ClosePool() {
	c.Close()
}

func (c *PGXPoolConn) Stats() sql.DBStats {
	stats := c.Stat()
	return sql.DBStats{
		MaxOpenConnections: int(stats.MaxConns()),
		OpenConnections:    int(stats.TotalConns()),
		InUse:              int(stats.AcquiredConns()),
		Idle:               int(stats.IdleConns()),
		// TODO: add wait stats (???)
		WaitCount:         0,
		WaitDuration:      0,
		MaxIdleClosed:     stats.MaxIdleDestroyCount(),
		MaxIdleTimeClosed: stats.MaxIdleDestroyCount(),
		MaxLifetimeClosed: stats.MaxLifetimeDestroyCount(),
	}
}

func newPgxPoolConn(
	ctx context.Context,
	cfg DBConfig,
	logger *zap.Logger,
	appName string,
) (*PGXPoolConn, error) {
	c, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgx pool parse config err: %w", err)
	}

	apmpgx.Instrument(c.ConnConfig)

	c.MinConns = int32(cfg.MaxIdleConnections)
	c.MaxConns = int32(cfg.MaxOpenConnections)
	c.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.ConnectConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("pgx pool connect config err: %w", err)
	}

	defer func() {
		if err != nil {
			pool.Close()
		}
	}()

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping pgx db err: %w", err)
	}

	conn := &PGXPoolConn{
		Pool:    pool,
		logger:  logger,
		appName: appName,
	}

	collector := sqlstats.NewStatsCollector(cfg.Name, conn)
	if err = prometheus.Register(collector); err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		return nil, fmt.Errorf("register pgx sqlstats err: %w", err)
	}

	return conn, nil
}
