package consul

import (
	"context"
	"fmt"
	"github.com/nenormalka/freya/conns/consul/sd"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/consul/client"
	"github.com/nenormalka/freya/conns/consul/config"
	"github.com/nenormalka/freya/conns/consul/kv"
	"github.com/nenormalka/freya/conns/consul/leader"
	"github.com/nenormalka/freya/conns/consul/lock"
	"github.com/nenormalka/freya/conns/consul/session"
	"github.com/nenormalka/freya/conns/consul/watcher"
)

type (
	Consul struct {
		cfg config.Config
		log *zap.Logger

		cli *api.Client
	}

	Watcher interface {
		Start(ctx context.Context) error
		Stop(ctx context.Context) error
		WatchKeys(keys watcher.WatchKeys) error
		WatchPrefixKeys(keys watcher.WatchPrefixKey) error
	}

	Locker interface {
		Acquire(ctx context.Context, key, sessionID string) (bool, error)
		Release(ctx context.Context, key, sessionID string) (bool, error)
		KeyOwner(ctx context.Context, key string) (string, error)
	}

	Session interface {
		Create(ctx context.Context) (string, error)
		Destroy(ctx context.Context) error
		Renew(ctx context.Context) <-chan error
		SessionID() string
		SessionKey() string
	}

	Leader interface {
		Start(ctx context.Context) error
		Stop(ctx context.Context) error
		IsLeader() bool
	}

	ServiceDiscovery interface {
		ServiceInfo(ctx context.Context, serviceName string, tags []string) ([]*api.ServiceEntry, error)
		ServiceList(ctx context.Context) (map[string][]string, error)
		ServiceRegister(ctx context.Context, reg *api.AgentServiceRegistration) error
		ServiceDeregister(ctx context.Context, serviceID string) error
	}
)

func NewConsul(cfg config.Config, logger *zap.Logger) (*Consul, error) {
	if cfg.Address == "" {
		return nil, nil
	}

	cli, err := client.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &Consul{
		cfg: cfg,
		log: logger,
		cli: cli,
	}, nil
}

func (c *Consul) Watcher() Watcher {
	return watcher.NewWatcher(c.cli, c.log)
}

func (c *Consul) Locker() Locker {
	return lock.NewLocker(c.cli)
}

func (c *Consul) KV() connectors.DBConnector[*api.KV, *api.Txn] {
	return kv.NewKV(c.cli)
}

func (c *Consul) Session() Session {
	return session.NewSession(c.cli, c.cfg)
}

func (c *Consul) Leader() Leader {
	return leader.NewLeader(
		lock.NewLocker(c.cli),
		session.NewSession(c.cli, c.cfg),
		watcher.NewWatcher(c.cli, c.log),
		c.log,
		c.cfg,
	)
}

func (c *Consul) ServiceDiscovery() ServiceDiscovery {
	return sd.NewServiceDiscovery(c.cli)
}
