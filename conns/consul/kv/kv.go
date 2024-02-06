package kv

import (
	"context"

	"github.com/hashicorp/consul/api"

	"github.com/nenormalka/freya/types"
)

type (
	KV struct {
		cli *api.Client
	}
)

func NewKV(cli *api.Client) *KV {
	return &KV{
		cli: cli,
	}
}

func (kv *KV) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, db *api.KV) error,
) error {
	return types.WithConsulKVMetrics(queryName, func() error {
		return callFunc(ctx, kv.cli.KV())
	})
}

func (kv *KV) CallTransaction(
	ctx context.Context,
	txName string,
	callFunc func(ctx context.Context, tx *api.Txn) error,
) error {
	return types.WithConsulKVMetrics(txName, func() error {
		return callFunc(ctx, kv.cli.Txn())
	})
}
