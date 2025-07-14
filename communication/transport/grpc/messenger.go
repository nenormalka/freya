package grpc

import (
	"context"

	"github.com/nenormalka/freya/communication/proto"
)

type (
	Messenger interface {
		Stream
		Close() error
	}

	Stream interface {
		Send(msg *proto.Message) error
		Recv() (*proto.Message, error)
		Context() context.Context
	}

	MessengerClient struct {
		stream proto.Transport_ProcessClient
	}

	MessengerServer struct {
		stream proto.Transport_ProcessServer
	}
)

func (mc *MessengerClient) Recv() (*proto.Message, error) {
	return mc.stream.Recv()
}

func (mc *MessengerClient) Send(msg *proto.Message) error {
	return mc.stream.Send(msg)
}

func (mc *MessengerClient) Close() error {
	return mc.stream.CloseSend()
}

func (mc *MessengerClient) Context() context.Context {
	return mc.stream.Context()
}

func (mr *MessengerServer) Recv() (*proto.Message, error) {
	return mr.stream.Recv()
}

func (mr *MessengerServer) Send(msg *proto.Message) error {
	return mr.stream.Send(msg)
}

func (mr *MessengerServer) Close() error {
	return nil
}

func (mr *MessengerServer) Context() context.Context {
	return mr.stream.Context()
}
