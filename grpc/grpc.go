package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/nenormalka/freya/types"

	"github.com/nenormalka/bishamon"
	"github.com/prometheus/client_golang/prometheus"
	"go.elastic.co/apm/v2"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/runtime/protoimpl"
)

type (
	Definition struct {
		Description    *grpc.ServiceDesc
		Implementation any
	}

	ServerOpt func(*Config)

	Params struct {
		dig.In

		GRPCDefinitions             []Definition                    `group:"grpc_impl"`
		GRPCUnaryCustomInterceptors [][]grpc.UnaryServerInterceptor `group:"grpc_unary_interceptor"`
		ServerOpt                   []ServerOpt                     `group:"grpc_server_opt"`
	}

	Server struct {
		cfg    *Config
		server *grpc.Server
		logger *zap.Logger
	}
)

func NewGRPC(
	p Params,
	cfg *Config,
	logger *zap.Logger,
	tracer *apm.Tracer,
) *Server {
	for _, opt := range p.ServerOpt {
		opt(cfg)
	}

	if cfg.WithServerMetrics {
		prometheus.MustRegister(types.ServerGRPCMetrics)
	}

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

	if cfg.WithServerMetrics {
		types.ServerGRPCMetrics.InitializeMetrics(grpcServer)
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

	return types.StartServerWithWaiting(ctx, s.logger, func(errCh chan error) {
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

func WithSensitiveData(sensitiveData *protoimpl.ExtensionInfo) (ServerOpt, error) {
	redactor, err := bishamon.NewClearRedactor(
		sensitiveData,
		bishamon.WithFieldsFromMapFunc(bishamon.CommonFieldsFromMapFunc),
	)
	if err != nil {
		return nil, fmt.Errorf("create redactor err: %w", err)
	}

	return func(cfg *Config) {
		cfg.LogRedactor = redactor
	}, nil
}
