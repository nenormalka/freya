package mocks

import (
	"context"
)

type (
	ConsulLeaderMock struct {
		isLeader bool
	}
)

func NewConsulLeaderMock(isLeader bool) *ConsulLeaderMock {
	return &ConsulLeaderMock{
		isLeader: isLeader,
	}
}

func (c *ConsulLeaderMock) IsLeader() bool {
	return c.isLeader
}

func (c *ConsulLeaderMock) Start(_ context.Context) error {
	return nil
}

func (c *ConsulLeaderMock) Stop(_ context.Context) error {
	return nil
}
