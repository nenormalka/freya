package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/nenormalka/freya/types"

	"go.elastic.co/apm/v2"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type (
	Definition struct {
		Description    *grpc.ServiceDesc
		Implementation interface{}
	}

	Params struct {
		dig.In

		GRPCDefinitions             []Definition                    `group:"grpc_impl"`
		GRPCUnaryCustomInterceptors [][]grpc.UnaryServerInterceptor `group:"grpc_unary_interceptor"`
	}

	Server struct {
		cfg    Config
		server *grpc.Server
		logger *zap.Logger
	}
)

func NewGRPC(
	p Params,
	cfg Config,
	logger *zap.Logger,
	tracer *apm.Tracer,
) *Server {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				Time:    cfg.KeepaliveTime,
				Timeout: cfg.KeepaliveTimeout,
			},
		),
		grpc.ChainUnaryInterceptor(interceptors(logger, tracer, p.GRPCUnaryCustomInterceptors, cfg)...),
	)

	p.GRPCDefinitions = append(p.GRPCDefinitions, Definition{
		Description:    &grpc_health_v1.Health_ServiceDesc,
		Implementation: health.NewServer(),
	})

	for _, def := range p.GRPCDefinitions {
		logger.Info(fmt.Sprintf("register grpc service: `%T`", def.Implementation))
		grpcServer.RegisterService(def.Description, def.Implementation)
	}

	if cfg.WithReflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		cfg:    cfg,
		server: grpcServer,
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("GRPC server started, listening on address: ", zap.String("grpc start", s.cfg.ListenAddr))

	return types.StartServerWithWaiting(ctx, func(errCh chan error) {
		listener, err := net.Listen("tcp", s.cfg.ListenAddr)
		if err != nil {
			s.logger.Error("create grpc listener err", zap.Error(err))
			if errCh != nil {
				errCh <- err
			}
		}

		if err = s.server.Serve(listener); err != nil {
			s.logger.Error("grpc server err", zap.Error(err))
			if errCh != nil {
				errCh <- err
			}
		}
	})
}

func (s *Server) Stop(_ context.Context) error {
	s.server.GracefulStop()
	return nil
}
