package conns

import (
	"errors"
	"github.com/doug-martin/goqu/v9"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jmoiron/sqlx"
	"github.com/nenormalka/freya/conns/connectors"
	"go.uber.org/zap"
)

type (
	Conns struct {
		elastic   *elasticsearch.Client
		logger    *zap.Logger
		poolDB    map[string]*sqlx.DB
		sqlConns  map[string]connectors.DBConnector[*sqlx.DB, *sqlx.Tx]
		goquConns map[string]connectors.DBConnector[*goqu.Database, *goqu.TxDatabase]
	}
)

var (
	errEmptyElasticConn = errors.New("empty elastic connection")
	errEmptyPool        = errors.New("empty pool")
	errEmptyConn        = errors.New("empty connection")
)

func NewConns(
	logger *zap.Logger,
	elastic *elasticsearch.Client,
	poolDB map[string]*sqlx.DB,
	sqlConns map[string]connectors.DBConnector[*sqlx.DB, *sqlx.Tx],
	goquConns map[string]connectors.DBConnector[*goqu.Database, *goqu.TxDatabase],
) *Conns {
	return &Conns{
		logger:    logger,
		elastic:   elastic,
		poolDB:    poolDB,
		sqlConns:  sqlConns,
		goquConns: goquConns,
	}
}

// Deprecated: Use GetSQLConnByName instead
func (c *Conns) GetDB() (*sqlx.DB, error) {
	return getConn[*sqlx.DB](c.poolDB, connectors.DefaultDBConn)
}

// Deprecated: Use GetSQLConnByName instead
func (c *Conns) GetSQLConn() (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error) {
	return c.GetSQLConnByName(connectors.DefaultDBConn)
}

func (c *Conns) GetSQLConnByName(nameConn string) (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error) {
	return getConn[connectors.DBConnector[*sqlx.DB, *sqlx.Tx]](c.sqlConns, nameConn)
}

// GetGoQuConn создает слой sql-builder'а для конструирования запросов в БД. Также он умеет делать scan в структуры
func (c *Conns) GetGoQuConn(nameConn string) (connectors.DBConnector[*goqu.Database, *goqu.TxDatabase], error) {
	return getConn[connectors.DBConnector[*goqu.Database, *goqu.TxDatabase]](c.goquConns, nameConn)
}

func (c *Conns) GetElastic() (*elasticsearch.Client, error) {
	if c.elastic == nil {
		return nil, errEmptyElasticConn
	}

	return c.elastic, nil
}

func (c *Conns) Close() {
	c.logger.Info("stopping connections")

	c.logger.Info("stop postgres")
	for i := range c.poolDB {
		if err := c.poolDB[i].Close(); err != nil {
			c.logger.Error("db stopping err", zap.Error(err))
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
