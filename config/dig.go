package config

import (
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: NewConfig},
	{CreateFunc: ConfigAdapter},
}

type (
	ConfigAdapterIn struct {
		dig.In

		CustomConfigurators []Configure `group:"configurators"`
	}
)

func ConfigAdapter(in ConfigAdapterIn) []Configure {
	loaders := []Configure{
		loadENV,
		loadYAML,
	}

	if len(in.CustomConfigurators) != 0 {
		loaders = append(loaders, in.CustomConfigurators...)
	}

	return loaders
}
