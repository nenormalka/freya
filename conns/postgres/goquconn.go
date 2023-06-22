package postrgres

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/types"
)

const (
	dialect = "postgres"
)

type (
	GoQuConn struct {
		db      *sqlx.DB
		logger  *zap.Logger
		appName string
	}
)

func NewGoQuConnector(
	poolDB map[string]*sqlx.DB,
	logger *zap.Logger,
	cfg *config.Config,
) map[string]connectors.DBConnector[*goqu.Database, *goqu.TxDatabase] {
	return newConns[connectors.DBConnector[*goqu.Database, *goqu.TxDatabase]](
		poolDB,
		func(nameConn string) connectors.DBConnector[*goqu.Database, *goqu.TxDatabase] {
			return &GoQuConn{
				db:      poolDB[nameConn],
				logger:  logger,
				appName: cfg.AppName,
			}
		})
}

func (s *GoQuConn) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, gq *goqu.Database) error,
) error {
	return types.WithSQLMetrics(queryName, s.appName, func() error {
		return callFunc(ctx, goqu.New(dialect, s.db))
	})
}

func (s *GoQuConn) CallTransaction(
	ctx context.Context,
	txName string,
	callFunc func(ctx context.Context, gqx *goqu.TxDatabase) error,
) error {
	return types.WithSQLMetrics(txName, s.appName, func() error {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}

		gqx := goqu.NewTx(dialect, tx)

		if err = callFunc(ctx, gqx); err != nil {
			if rErr := gqx.Rollback(); rErr != nil {
				s.logger.Error(fmt.Sprintf("failed rollback transaction: %s", txName), zap.Error(rErr))
			}

			return err
		}

		return gqx.Commit()
	})
}
