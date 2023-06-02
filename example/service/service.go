package service

import (
	"context"
	"fmt"

	"github.com/nenormalka/freya/example/repo"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
	"go.uber.org/zap"
)

var Module = types.Module{
	{CreateFunc: NewService},
	{CreateFunc: Adapter},
	{CreateFunc: AdapterParam},
}

type (
	AdapterOut struct {
		dig.Out

		Service types.Runnable `group:"services"`
	}

	AdapterParams struct {
		dig.In

		Repo   *repo.Repo
		Logger *zap.Logger
	}
)

func Adapter(s *Service) AdapterOut {
	return AdapterOut{
		Service: s,
	}
}

func AdapterParam(in AdapterParams) Params {
	return Params{
		Repo:   in.Repo,
		Logger: in.Logger,
	}
}

type (
	Params struct {
		Repo   Repo
		Logger *zap.Logger
	}

	Repo interface {
		GetNow(ctx context.Context) (string, error)
	}

	Service struct {
		logger *zap.Logger
		repo   Repo
	}
)

func NewService(p Params) *Service {
	return &Service{
		logger: p.Logger,
		repo:   p.Repo,
	}
}

func (s *Service) Start(ctx context.Context) error {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("get now from repo err: %w", err)
	}

	s.logger.Info("service run at", zap.String("time", now))

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("start service err: %w", err)
	}

	s.logger.Info("service stopped at", zap.String("time", now))

	return nil
}

func (s *Service) Now(ctx context.Context) (string, error) {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return "", fmt.Errorf("get now from repo err: %w", err)
	}

	return now, nil
}
