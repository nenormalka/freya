package connectors

import (
	"context"

	txtype "github.com/nenormalka/freya/conns/couchbase/types"
	"github.com/nenormalka/freya/conns/postgres/types"
	redisClient "github.com/nenormalka/freya/conns/redis/types"

	"github.com/couchbase/gocb/v2"
	"github.com/doug-martin/goqu/v9"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/hashicorp/consul/api"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type (
	ConnectDB interface {
		*sqlx.DB | *goqu.Database | types.PgxConn | *gocb.Collection | *elasticsearch.Client | *api.KV | *redis.Client
	}

	ConnectTx interface {
		*sqlx.Tx | *goqu.TxDatabase | types.PgxTx | *txtype.CollectionTx | *api.Txn | *redisClient.RedisTx
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
	SlaveDBConn   = "slave"
)

const (
	PgxConnType  = "pgx"
	SqlxConnType = "sqlx"
)

const (
	PostgresDBType = "pgx"
	MsSQLDBType    = "mssql"
)
