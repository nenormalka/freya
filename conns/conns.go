package conns

import (
	"errors"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jmoiron/sqlx"
	"github.com/nenormalka/freya/conns/connectors"
	"go.uber.org/zap"
)

type (
	Conns struct {
		postgres *sqlx.DB
		elastic  *elasticsearch.Client
		logger   *zap.Logger
		sqlConn  connectors.SQLConnector
		goquConn connectors.GoQuConnector
	}
)

var (
	errEmptyDBConn      = errors.New("empty db connection")
	errEmptySQLConn     = errors.New("empty sql connection")
	errEmptyElasticConn = errors.New("empty elastic connection")
)

func NewConns(
	postgres *sqlx.DB,
	elastic *elasticsearch.Client,
	logger *zap.Logger,
	sqlConn connectors.SQLConnector,
	goquConn connectors.GoQuConnector,
) *Conns {
	return &Conns{
		postgres: postgres,
		elastic:  elastic,
		logger:   logger,
		sqlConn:  sqlConn,
		goquConn: goquConn,
	}
}

// Deprecated: Use GetSQLConn
func (c *Conns) GetDB() (*sqlx.DB, error) {
	if c.postgres == nil {
		return nil, errEmptyDBConn
	}

	return c.postgres, nil
}

func (c *Conns) GetSQLConn() (connectors.SQLConnector, error) {
	if c.sqlConn == nil {
		return nil, errEmptySQLConn
	}

	return c.sqlConn, nil
}

// GetGoQuConn создает слой sql-builder'а для конструирования запросов в БД. Также он умеет делать scan в структуры
func (c *Conns) GetGoQuConn() (connectors.GoQuConnector, error) {
	if c.goquConn == nil {
		return nil, errEmptyDBConn
	}

	return c.goquConn, nil
}

func (c *Conns) GetElastic() (*elasticsearch.Client, error) {
	if c.elastic == nil {
		return nil, errEmptyElasticConn
	}

	return c.elastic, nil
}

func (c *Conns) Close() {
	c.logger.Info("stopping connections")

	if c.postgres != nil {
		c.logger.Info("stop postgres")

		if err := c.postgres.Close(); err != nil {
			c.logger.Error("postgres stopping err", zap.Error(err))
		}
	}

	// stop other connections
}
