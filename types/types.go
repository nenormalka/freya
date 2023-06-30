package types

import (
	"context"
	"go.uber.org/dig"
)

type (
	Provider struct {
		CreateFunc any
		Options    []dig.ProvideOption
	}

	Module []Provider

	App interface {
		Run(context.Context) error
	}
)

func (m Module) Append(o Module) Module {
	return append(m, o...)
}
