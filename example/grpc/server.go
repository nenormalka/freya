package example_service

import (
	"context"

	"freya/example/service"

	"github.com/nenormalka/freya/types/errors"

	"go.uber.org/zap"
)

type (
	Server struct {
		logger  *zap.Logger
		service *service.Service

		UnimplementedExampleServiceServer
	}
)

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
		return nil, err
	}

	s.logger.Info("request time", zap.String("time", now))

	return &Empty{}, nil
}

func (s *Server) GetErr(_ context.Context, req *GetErrRequest) (*Empty, error) {
	return nil, s.service.GetErr(errors.Code(req.Code))
}

func (s *Server) GetSensitive(_ context.Context, req *GetSensitiveRequest) (*GetSensitiveResponse, error) {
	return &GetSensitiveResponse{
		Data: map[string]string{
			"login ":   req.Login,
			"password": req.Password,
		},
	}, nil
}
