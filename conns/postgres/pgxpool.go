package postrgres

import (
	"context"
	"fmt"

	"github.com/nenormalka/freya/conns/connectors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.elastic.co/apm/module/apmpgx/v2"
)

func NewPGXPool(
	ctx context.Context,
	config PostgresConfig,
) (map[string]*pgxpool.Pool, error) {
	if len(config.Configs) == 0 {
		return nil, nil
	}

	poolDB := make(map[string]*pgxpool.Pool)

	for i := range config.Configs {
		if config.Configs[i].Type != connectors.PgxConnType {
			continue
		}

		pool, err := newPgxPool(ctx, config.Configs[i])
		if err != nil {
			return nil, fmt.Errorf("new pgx pool err: %w", err)
		}

		poolDB[config.Configs[i].Name] = pool
	}

	return poolDB, nil
}

func newPgxPool(
	ctx context.Context,
	cfg DBConfig,
) (*pgxpool.Pool, error) {
	c, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgx pool parse config err: %w", err)
	}

	apmpgx.Instrument(c.ConnConfig)

	c.MinConns = int32(cfg.MaxIdleConnections)
	c.MaxConns = int32(cfg.MaxOpenConnections)
	c.MaxConnLifetime = cfg.ConnMaxLifetime

	c.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		cfg.PreferSimpleProtocol = true
		return nil
	}

	pool, err := pgxpool.ConnectConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("pgx pool connect config err: %w", err)
	}

	defer func() {
		if err != nil {
			pool.Close()
		}
	}()

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping pgx db err: %w", err)
	}

	return pool, nil
}
