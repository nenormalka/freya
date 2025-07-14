package service

import (
	"context"
	"errors"
	"fmt"

	ferrors "github.com/nenormalka/freya/types/errors"

	"go.uber.org/zap"
)

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

func NewService(p Params) (*Service, error) {
	return &Service{
		logger: p.Logger,
		repo:   p.Repo,
	}, nil
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
	var now string
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return "", fmt.Errorf("get now from repo err: %w", err)
	}

	return now, nil
}

func (s *Service) GetErr(code ferrors.Code) error {
	switch code {
	case ferrors.InvalidArgument:
		return fmt.Errorf("err %w", ferrors.NewInvalidError(errors.New("invalid argument")))
	case ferrors.NotFound:
		return ferrors.NewNotFoundError(errors.New("not found")).AddDetail("id", "2")
	default:
		return ferrors.NewUnknownError(errors.New("unknown error")).AddDetail("id", "3")
	}
}
