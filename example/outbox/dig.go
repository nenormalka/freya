package outbox

import (
	"github.com/nenormalka/melissa/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: NewOutbox},
	{CreateFunc: Adapter},
}

type (
	AdapterOut struct {
		dig.Out

		Outbox types.Runnable `group:"services"`
	}
)

func Adapter(o *Outbox) AdapterOut {
	return AdapterOut{
		Outbox: o,
	}
}
