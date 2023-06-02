package config

import (
	"fmt"

	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: Adapter},
}

type (
	AdapterOut struct {
		dig.Out

		Configurator config.Configurator `group:"configurators"`
	}
)

func Adapter() AdapterOut {
	return AdapterOut{
		Configurator: Configure,
	}
}

func Configure(_ *config.Config) error {
	fmt.Println("custom configuring...")
	return nil
}
