package grpc

import (
	"github.com/nenormalka/melissa/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: NewGRPCConfig},
	{CreateFunc: NewGRPC},
	{CreateFunc: Adapter},
	{CreateFunc: newServersHelper},
}

type (
	AdapterOut struct {
		dig.Out

		Server types.Runnable `group:"servers"`
	}

	AdapterIn struct {
		dig.In

		Server *Server
	}
)

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Server: in.Server,
	}
}
