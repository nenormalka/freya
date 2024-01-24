package http

import (
	"github.com/nenormalka/freya/http"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
)

var Module = types.Module{
	{CreateFunc: newHTTP},
	{CreateFunc: adapter},
}

type (
	AdapterOut struct {
		dig.Out

		Server http.CustomServer `group:"custom_http_servers"`
	}
)

// adapter ...
func adapter(serv *Server) AdapterOut {
	return AdapterOut{
		Server: serv,
	}
}
