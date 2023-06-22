package postrgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/types"
)

type (
	SQLConn struct {
		db      *sqlx.DB
		logger  *zap.Logger
		appName string
	}
)

func NewSQLConnector(
	poolDB map[string]*sqlx.DB,
	logger *zap.Logger,
	cfg *config.Config,
) map[string]connectors.DBConnector[*sqlx.DB, *sqlx.Tx] {
	return newConns[connectors.DBConnector[*sqlx.DB, *sqlx.Tx]](
		poolDB,
		func(nameConn string) connectors.DBConnector[*sqlx.DB, *sqlx.Tx] {
			return &SQLConn{
				db:      poolDB[nameConn],
				logger:  logger,
				appName: cfg.AppName,
			}
		})
}

func (s *SQLConn) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, db *sqlx.DB) error,
) error {
	return types.WithSQLMetrics(queryName, s.appName, func() error {
		return callFunc(ctx, s.db)
	})
}

func (s *SQLConn) CallTransaction(
	ctx context.Context,
	txName string,
	callFunc func(ctx context.Context, tx *sqlx.Tx) error,
) error {
	return types.WithSQLMetrics(txName, s.appName, func() error {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}

		if err = callFunc(ctx, tx); err != nil {
			if rErr := tx.Rollback(); rErr != nil {
				s.logger.Error(fmt.Sprintf("failed rollback transaction: %s", txName), zap.Error(rErr))
			}

			return err
		}

		return tx.Commit()
	})
}
