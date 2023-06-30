package types

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
)

type (
	PgxQuerier interface {
		pgxtype.Querier
		Get(ctx context.Context, dst any, query string, args ...any) error
		Select(ctx context.Context, dst any, query string, args ...any) error
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

func (p *PgxTx) Select(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Select(ctx, p, dst, query, args...)
}

func (p *PgxTx) Get(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Get(ctx, p, dst, query, args...)
}
