package http

import (
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: NewHTTPConfig},
	{CreateFunc: NewHTTP},
	{CreateFunc: Adapter},
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

	CustomServerList struct {
		dig.In

		CustomServers []CustomServer `group:"custom_http_servers"`
	}
)

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Server: in.Server,
	}
}
