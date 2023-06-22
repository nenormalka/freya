package connectors

import (
	"context"

	"github.com/nenormalka/freya/conns/postgres/types"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
)

type (
	ConnectDB interface {
		*sqlx.DB | *goqu.Database | types.PgxConn
	}

	ConnectTx interface {
		*sqlx.Tx | *goqu.TxDatabase | types.PgxTx
	}

	DBConnector[T ConnectDB, M ConnectTx] interface {
		CallContext(
			ctx context.Context,
			queryName string,
			callFunc func(ctx context.Context, db T) error,
		) error
		CallTransaction(
			ctx context.Context,
			txName string,
			callFunc func(ctx context.Context, tx M) error,
		) error
	}
)

const (
	DefaultDBConn = "master"
)

const (
	PgxConnType  = "pgx"
	SqlxConnType = "sqlx"
)
