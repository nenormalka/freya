package types

import (
	"go.uber.org/dig"
)

type (
	Provider struct {
		CreateFunc any
		Options    []dig.ProvideOption
	}

	Module []Provider
)

func (m Module) Append(o Module) Module {
	return append(m, o...)
}
