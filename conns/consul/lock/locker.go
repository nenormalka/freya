package lock

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
)

type (
	Locker struct {
		cli *api.Client
	}
)

func NewLocker(cli *api.Client) *Locker {
	return &Locker{cli: cli}
}

func (l *Locker) Acquire(ctx context.Context, key, sessionID string) (bool, error) {
	opt := &api.WriteOptions{}

	acquired, _, err := l.cli.KV().Acquire(&api.KVPair{
		Key:     key,
		Session: sessionID,
		Value:   []byte(sessionID),
	}, opt.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("failed to acquire key %s: sessionID %s: %w", key, sessionID, err)
	}

	return acquired, nil
}

func (l *Locker) Release(ctx context.Context, key, sessionID string) (bool, error) {
	opt := &api.WriteOptions{}

	release, _, err := l.cli.KV().Release(&api.KVPair{
		Key:     key,
		Session: sessionID,
	}, opt.WithContext(ctx))
	if err != nil {
		return release, fmt.Errorf("failed to release key %s: sessionID %s: %w", key, sessionID, err)
	}

	return release, nil
}

func (l *Locker) KeyOwner(ctx context.Context, key string) (string, error) {
	opt := &api.QueryOptions{}

	kv, _, err := l.cli.KV().Get(key, opt.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if kv == nil {
		return "", nil
	}

	return kv.Session, nil
}
