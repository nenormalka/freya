package mocks

import (
	"context"

	redistypes "github.com/nenormalka/freya/conns/redis/types"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

type (
	RedisMock struct {
		client *redis.Client
		Mock   redismock.ClientMock
	}
)

func NewRedisMock() *RedisMock {
	db, m := redismock.NewClientMock()

	return &RedisMock{
		Mock:   m,
		client: db,
	}
}

func (r *RedisMock) CallContext(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, client *redis.Client) error,
) error {
	return callFunc(ctx, r.client)
}

func (r *RedisMock) CallTransaction(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, tx *redistypes.RedisTx) error,
) error {
	return callFunc(ctx, redistypes.NewRedisTx(r.client.Watch))
}
