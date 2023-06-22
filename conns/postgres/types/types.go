package types

import (
	"context"

	"github.com/jackc/pgtype/pgxtype"
)

type (
	PgxQuerier interface {
		pgxtype.Querier
		Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error
		Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	}

	PgxConn struct {
		PgxQuerier
	}
)
