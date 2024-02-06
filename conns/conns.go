package conns

import (
	"errors"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/consul"
	"github.com/nenormalka/freya/conns/couchbase"
	"github.com/nenormalka/freya/conns/elastic"
	"github.com/nenormalka/freya/conns/kafka"
	dbtypes "github.com/nenormalka/freya/conns/postgres/types"

	"github.com/doug-martin/goqu/v9"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type (
	Conns struct {
		// logger логгер
		logger *zap.Logger
		// elastic клиент для работы с эластиком
		// Deprecated: elasticConn
		elastic *elasticsearch.Client
		// elastic клиент для работы с эластиком c метриками
		elasticConn *elastic.ElasticConn
		// db_name -> коннект, мапа с соединениями к бд, нужна, чтобы закрыть всё при выключении сервиса
		sqlxPoolDB map[string]*sqlx.DB
		// db_name -> пул коннектов pgx, мапа с соединениями к бд, нужна, чтобы закрыть всё при выключении сервиса
		pgxPoolDB map[string]*pgxpool.Pool
		// DBConnector обёртка нужна, чтобы собирать метрики в прометеус через WithSQLMetrics
		// db_name -> обёртка над sqlx
		sqlConns map[string]connectors.DBConnector[*sqlx.DB, *sqlx.Tx]
		// db_name -> обёртка над goqu
		goquConns map[string]connectors.DBConnector[*goqu.Database, *goqu.TxDatabase]
		// db_name -> обёртка над pgx
		pgxConns map[string]connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx]
		// kafka абстракция над кафкой
		kafka *kafka.Kafka
		// couchbase абстракция над коучбейсом
		couchbase *couchbase.Couchbase
		// consul абстракция над консулом
		consul *consul.Consul
	}
)

var (
	errEmptyElasticConn = errors.New("empty elastic connection")
	errEmptyPool        = errors.New("empty pool")
	errEmptyConn        = errors.New("empty connection")
	errEmptyKafka       = errors.New("empty kafka")
	errEmptyConsul      = errors.New("empty consul")
	errEmptyCouchbase   = errors.New("empty couchbase")
)

func NewConns(
	logger *zap.Logger,
	elastic *elasticsearch.Client,
	elasticConn *elastic.ElasticConn,
	sqlxPoolDB map[string]*sqlx.DB,
	pgxPoolDB map[string]*pgxpool.Pool,
	sqlConns map[string]connectors.DBConnector[*sqlx.DB, *sqlx.Tx],
	goquConns map[string]connectors.DBConnector[*goqu.Database, *goqu.TxDatabase],
	pgxConns map[string]connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx],
	kafka *kafka.Kafka,
	couchbase *couchbase.Couchbase,
	consul *consul.Consul,
) *Conns {
	return &Conns{
		logger:      logger,
		elastic:     elastic,
		elasticConn: elasticConn,
		sqlxPoolDB:  sqlxPoolDB,
		sqlConns:    sqlConns,
		goquConns:   goquConns,
		pgxConns:    pgxConns,
		pgxPoolDB:   pgxPoolDB,
		kafka:       kafka,
		couchbase:   couchbase,
		consul:      consul,
	}
}

// Deprecated: Use GetSQLConnByName instead
func (c *Conns) GetDB() (*sqlx.DB, error) {
	return getConn[*sqlx.DB](c.sqlxPoolDB, connectors.DefaultDBConn)
}

// Deprecated: Use GetSQLConnByName instead
func (c *Conns) GetSQLConn() (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error) {
	return c.GetSQLConnByName(connectors.DefaultDBConn)
}

// GetSQLConnByName возвращает коннект к бд, если он есть
func (c *Conns) GetSQLConnByName(nameConn string) (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error) {
	return getConn[connectors.DBConnector[*sqlx.DB, *sqlx.Tx]](c.sqlConns, nameConn)
}

// GetPGXConnByName возвращает коннект к бд, если он есть
func (c *Conns) GetPGXConnByName(nameConn string) (connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx], error) {
	return getConn[connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx]](c.pgxConns, nameConn)
}

// GetGoQuConn создает слой sql-builder'а для конструирования запросов в БД. Также он умеет делать scan в структуры
func (c *Conns) GetGoQuConn(nameConn string) (connectors.DBConnector[*goqu.Database, *goqu.TxDatabase], error) {
	return getConn[connectors.DBConnector[*goqu.Database, *goqu.TxDatabase]](c.goquConns, nameConn)
}

// GetKafka возвращает абстракцию над кафкой
func (c *Conns) GetKafka() (*kafka.Kafka, error) {
	if c.kafka == nil {
		return nil, errEmptyKafka
	}

	return c.kafka, nil
}

// GetConsul возвращает абстракцию над консулом
func (c *Conns) GetConsul() (*consul.Consul, error) {
	if c.consul == nil {
		return nil, errEmptyConsul
	}

	return c.consul, nil
}

// GetElastic возвращает клиент для работы с эластиком
// Deprecated: Use GetElasticConn instead
func (c *Conns) GetElastic() (*elasticsearch.Client, error) {
	if c.elastic == nil {
		return nil, errEmptyElasticConn
	}

	return c.elastic, nil
}

func (c *Conns) GetElasticConn() (*elastic.ElasticConn, error) {
	if c.elasticConn == nil {
		return nil, errEmptyElasticConn
	}

	return c.elasticConn, nil
}

// GetCouchbase возвращает абстракцию над коучбейсом
func (c *Conns) GetCouchbase() (*couchbase.Couchbase, error) {
	if c.couchbase == nil {
		return nil, errEmptyCouchbase
	}

	return c.couchbase, nil
}

func (c *Conns) Close() {
	c.logger.Info("stopping connections")

	if len(c.sqlxPoolDB) != 0 {
		c.logger.Info("stop sqlx connections")
		for i := range c.sqlxPoolDB {
			if err := c.sqlxPoolDB[i].Close(); err != nil {
				c.logger.Error("db stopping err", zap.Error(err))
			}
		}
	}

	if len(c.pgxPoolDB) != 0 {
		c.logger.Info("stop pgx connections")
		for i := range c.pgxPoolDB {
			c.pgxPoolDB[i].Close()
		}
	}

	// stop other connections
}

func getConn[T any](m map[string]T, name string) (T, error) {
	var t T

	if len(m) == 0 {
		return t, errEmptyPool
	}

	if conn, ok := m[name]; ok {
		return conn, nil

	}

	return t, errEmptyConn
}
