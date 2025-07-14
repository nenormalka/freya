package redis

import (
	"context"
	"fmt"

	"github.com/nenormalka/freya/conns/redis/config"
	redistypes "github.com/nenormalka/freya/conns/redis/types"
	"github.com/nenormalka/freya/types"

	"github.com/redis/go-redis/v9"
)

type (
	RedisClient struct {
		client *redis.Client
	}
)

func NewRedisClient(config *config.Config) (*RedisClient, error) {
	if config.DSN == "" {
		return nil, nil
	}

	cfg, err := redis.ParseURL(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis dsn: %v", err)
	}

	cfg.PoolSize = config.PoolSize

	client := redis.NewClient(cfg)

	if status := client.Ping(context.Background()); status.Err() != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", status.Err())
	}

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, client *redis.Client) error,
) error {
	return types.WithRedisMetrics(queryName, func() error {
		return callFunc(ctx, r.client)
	})
}

func (r *RedisClient) CallTransaction(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, client *redistypes.RedisTx) error,
) error {
	return types.WithRedisMetrics(queryName, func() error {
		return callFunc(ctx, redistypes.NewRedisTx(r.client.Watch))
	})
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
