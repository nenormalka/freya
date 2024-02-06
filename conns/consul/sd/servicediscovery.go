package sd

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
)

type (
	ServiceDiscovery struct {
		cli *api.Client
	}
)

func NewServiceDiscovery(cli *api.Client) *ServiceDiscovery {
	return &ServiceDiscovery{cli: cli}
}

func (s *ServiceDiscovery) ServiceRegister(ctx context.Context, reg *api.AgentServiceRegistration) error {
	opts := api.ServiceRegisterOpts{}
	opts.WithContext(ctx)

	if err := s.cli.Agent().ServiceRegisterOpts(reg, opts); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

func (s *ServiceDiscovery) ServiceDeregister(ctx context.Context, serviceID string) error {
	opts := &api.QueryOptions{}
	opts.WithContext(ctx)

	if err := s.cli.Agent().ServiceDeregisterOpts(serviceID, opts); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	return nil
}

func (s *ServiceDiscovery) ServiceInfo(ctx context.Context, serviceName string, tags []string) ([]*api.ServiceEntry, error) {
	opts := &api.QueryOptions{}
	opts.WithContext(ctx)

	services, _, err := s.cli.Health().ServiceMultipleTags(serviceName, tags, true, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return services, nil
}

func (s *ServiceDiscovery) ServiceList(ctx context.Context) (map[string][]string, error) {
	opts := &api.QueryOptions{}
	opts.WithContext(ctx)

	res, _, err := s.cli.Catalog().Services(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	return res, nil
}
