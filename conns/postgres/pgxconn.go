package postrgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/postgres/collector"
	dbtypes "github.com/nenormalka/freya/conns/postgres/types"
	"github.com/nenormalka/freya/types"

	"github.com/nenormalka/freya/config"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type (
	PGXPoolConn struct {
		*pgxpool.Pool
		logger  *zap.Logger
		appName string
	}
)

func NewPGXPoolConn(
	pgxPool map[string]*pgxpool.Pool,
	logger *zap.Logger,
	cfg *config.Config,
) (map[string]connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx], error) {
	if len(pgxPool) == 0 {
		return nil, nil
	}

	pools := make(map[string]connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx], len(pgxPool))

	for i := range pgxPool {
		conn := &PGXPoolConn{
			Pool:    pgxPool[i],
			logger:  logger,
			appName: cfg.AppName,
		}

		if err := collector.CollectDBStats(i, conn); err != nil {
			return nil, fmt.Errorf("failed to collect db stats: %w", err)
		}

		pools[i] = conn
	}

	return pools, nil
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
	callFunc func(ctx context.Context, tx dbtypes.PgxTx) error,
) error {
	return types.WithSQLMetrics(txName, c.appName, func() error {
		tx, err := c.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}

		if err = callFunc(ctx, dbtypes.PgxTx{PgxTransactor: tx}); err != nil {
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
