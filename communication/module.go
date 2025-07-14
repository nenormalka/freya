package communication

import (
	"github.com/nenormalka/freya/communication/model"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/melissa/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: NewCommunication},
	{CreateFunc: Adapter},
	{CreateFunc: NewParams},
	{CreateFunc: model.NewConfig},
}

type (
	AdapterOut struct {
		dig.Out

		Service types.Runnable `group:"services"`
	}

	AdapterIn struct {
		dig.In

		Connections   *conns.Conns
		ServersHelper *grpc.ServersHelper
	}
)

func Adapter(c Communicator) AdapterOut {
	return AdapterOut{
		Service: c,
	}
}

func NewParams(in AdapterIn) *Params {
	return &Params{
		Connections:   in.Connections,
		ServersHelper: in.ServersHelper,
	}
}
