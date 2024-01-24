package connectors

import (
	"context"

	"github.com/elastic/go-elasticsearch/v8"
	txtype "github.com/nenormalka/freya/conns/couchbase/types"
	"github.com/nenormalka/freya/conns/postgres/types"

	"github.com/couchbase/gocb/v2"
	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
)

type (
	ConnectDB interface {
		*sqlx.DB | *goqu.Database | types.PgxConn | *gocb.Collection | *elasticsearch.Client
	}

	ConnectTx interface {
		*sqlx.Tx | *goqu.TxDatabase | types.PgxTx | *txtype.CollectionTx
	}

	CallContextConnector[T ConnectDB] interface {
		CallContext(
			ctx context.Context,
			queryName string,
			callFunc func(ctx context.Context, db T) error,
		) error
	}

	CallTransactionConnector[M ConnectTx] interface {
		CallTransaction(
			ctx context.Context,
			txName string,
			callFunc func(ctx context.Context, tx M) error,
		) error
	}

	DBConnector[T ConnectDB, M ConnectTx] interface {
		CallContextConnector[T]
		CallTransactionConnector[M]
	}
)

const (
	DefaultDBConn = "master"
)

const (
	PgxConnType  = "pgx"
	SqlxConnType = "sqlx"
)
