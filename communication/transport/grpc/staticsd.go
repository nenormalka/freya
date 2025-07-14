package grpc

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidServiceName = errors.New("invalid service name")
	ErrInvalidServices    = errors.New("invalid service")
	ErrEmptyAddress       = errors.New("empty address")
)

type (
	StaticServiceDiscovery struct {
		Addresses map[string][]string
	}
)

func NewStaticServiceDiscovery(addresses string) (*StaticServiceDiscovery, error) {
	addrs, err := getAddresses(addresses)
	if err != nil {
		return nil, fmt.Errorf("get addresses err: %w", err)
	}

	return &StaticServiceDiscovery{
		Addresses: addrs,
	}, nil
}

func (s *StaticServiceDiscovery) GetServicesByServiceName(serviceName string) ([]string, error) {
	addrs, ok := s.Addresses[serviceName]
	if !ok {
		return nil, ErrInvalidServiceName
	}

	return addrs, nil
}

func getAddresses(addrs string) (map[string][]string, error) {
	addresses := make(map[string][]string)

	if addrs == "" {
		return addresses, nil
	}

	services := strings.Split(addrs, ";")

	for _, service := range services {
		parts := strings.Split(service, "=")
		if len(parts) != 2 {
			return nil, ErrInvalidServices
		}

		serviceName := parts[0]
		serviceAddrs := strings.Split(parts[1], ",")
		for _, addr := range serviceAddrs {
			if addr == "" {
				return nil, ErrEmptyAddress
			}
		}

		addresses[serviceName] = serviceAddrs
	}

	return addresses, nil
}
