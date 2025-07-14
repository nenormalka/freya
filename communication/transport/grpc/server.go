package grpc

import (
	"errors"

	"github.com/nenormalka/freya/communication/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	Server struct {
		proto.UnimplementedTransportServer

		handler func(stream proto.Transport_ProcessServer) error
	}
)

func newServer(handler func(stream proto.Transport_ProcessServer) error) *Server {
	return &Server{
		handler: handler,
	}
}

func (s *Server) Process(stream proto.Transport_ProcessServer) error {
	if err := stream.Context().Err(); err != nil {
		return status.Error(codes.Canceled, err.Error())
	}

	if err := s.handler(stream); err != nil {
		switch {
		case errors.Is(err, ErrItsMeMario):
			return status.Error(codes.Code(99999), err.Error())
		case errors.Is(err, ErrServiceExists):
			return status.Error(codes.AlreadyExists, err.Error())
		default:
			return status.Error(codes.Internal, err.Error())
		}
	}

	return nil
}
