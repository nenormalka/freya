package connectors

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
)

type (
	SQLConnector interface {
		CallContext(
			ctx context.Context,
			queryName string,
			callFunc func(ctx context.Context, db *sqlx.DB) error,
		) error
		CallTransaction(
			ctx context.Context,
			txName string,
			callFunc func(ctx context.Context, tx *sqlx.Tx) error,
		) error
	}

	GoQuConnector interface {
		CallContext(
			ctx context.Context,
			queryName string,
			callFunc func(ctx context.Context, gq *goqu.Database) error,
		) error

		CallTransaction(
			ctx context.Context,
			txName string,
			callFunc func(ctx context.Context, gqx *goqu.TxDatabase) error,
		) error
	}
)

const (
	DefaultDBConn = "master"
)
