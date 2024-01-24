package service

import (
	"freya/example/repo"

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
