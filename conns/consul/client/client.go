package client

import (
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"

	"github.com/nenormalka/freya/conns/consul/config"
)

var (
	ErrEmptyPeers = errors.New("consul peers is empty")
)

func NewClient(cfg config.Config) (*api.Client, error) {
	conf := api.DefaultConfig()
	conf.Address = cfg.Address
	conf.Token = cfg.Token
	conf.Scheme = cfg.Scheme
	conf.TLSConfig = api.TLSConfig{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	consul, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	peers, err := consul.Status().Peers()
	if err != nil {
		return nil, fmt.Errorf("failed to get consul peers: %w", err)
	}

	if len(peers) == 0 {
		return nil, ErrEmptyPeers
	}

	return consul, nil
}
