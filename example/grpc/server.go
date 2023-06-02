package example_service

import (
	"context"
	"fmt"
	"time"

	"github.com/nenormalka/freya/example/service"
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/freya/metadata"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
	"go.uber.org/zap"
	grpc2 "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	}
}

func getInterceptors() []grpc2.UnaryServerInterceptor {
	return []grpc2.UnaryServerInterceptor{
		getCustomInterceptor(),
		printMetadataInterceptor(),
	}
}

func printMetadataInterceptor() grpc2.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc2.UnaryServerInfo,
		handler grpc2.UnaryHandler,
	) (interface{}, error) {

		res := ""

		res, err := metadata.GetAppVersion(ctx)
		if err != nil {
			res = err.Error()
		}
		fmt.Println("md app version", res)

		res, err = metadata.GetPlatform(ctx)
		if err != nil {
			res = err.Error()
		}
		fmt.Println("md platform", res)

		return handler(ctx, req)
	}
}

func getCustomInterceptor() grpc2.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc2.UnaryServerInfo,
		handler grpc2.UnaryHandler,
	) (interface{}, error) {
		fmt.Println(">>> ", time.Now().String())
		resp, err := handler(ctx, req)
		fmt.Println("<<< ", time.Now().String())
		return resp, err
	}
}

type Server struct {
	logger  *zap.Logger
	service *service.Service

	UnsafeExampleServiceServer
}

func newGRPC(
	logger *zap.Logger,
	service *service.Service,
) *Server {
	return &Server{
		logger:  logger,
		service: service,
	}
}

func (s *Server) GetTest(ctx context.Context, _ *Empty) (*Empty, error) {
	now, err := s.service.Now(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.logger.Info("request time", zap.String("time", now))

	return &Empty{}, nil
}
