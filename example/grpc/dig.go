package example_service

import (
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
	grpc2 "google.golang.org/grpc"
)

var Module = types.Module{
	{CreateFunc: newGRPC},
	{CreateFunc: adapter},
}

type (
	AdapterOut struct {
		dig.Out

		Server       grpc.Definition                `group:"grpc_impl"`
		Interceptors []grpc2.UnaryServerInterceptor `group:"grpc_unary_interceptor"`
		ServerOpt    grpc.ServerOpt                 `group:"grpc_server_opt"`
	}
)

// adapter ...
func adapter(serv *Server) AdapterOut {
	return AdapterOut{
		Server: grpc.Definition{
			Description:    &ExampleService_ServiceDesc,
			Implementation: serv,
		},
		Interceptors: getInterceptors(),
		ServerOpt:    grpc.WithSensitiveData(E_Sensitive),
	}
}
