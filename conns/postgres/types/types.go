package types

import (
	"context"
	"github.com/jackc/pgx/v4"

	"github.com/jackc/pgtype/pgxtype"
)

type (
	PgxQuerier interface {
		pgxtype.Querier
		Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error
		Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	}

	PgxTransactor interface {
		pgx.Tx
	}

	PgxConn struct {
		PgxQuerier
	}

	PgxTx struct {
		PgxTransactor
	}
)
