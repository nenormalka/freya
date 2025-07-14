package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/nenormalka/freya/communication/model"
	"github.com/nenormalka/freya/communication/proto"
	helper "github.com/nenormalka/freya/grpc"

	"github.com/google/uuid"

	"go.elastic.co/apm/module/apmgrpc/v2"

	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	discoveryTime = 20 * time.Second
)

var (
	ErrItsMeMario    = errors.New("it's me, Mario")
	ErrServiceExists = errors.New("service exists")
	ErrMissHandshake = errors.New("miss handshake")
	ErrSkipConnect   = errors.New("skip connect")
)

type (
	ServiceDiscovery interface {
		GetServicesByServiceName(serviceName string) ([]string, error)
	}

	Service struct {
		logger *zap.Logger

		clientCode  string
		serviceName string
		peers       map[string]Messenger
		muPeers     sync.Mutex
		sd          ServiceDiscovery
		handler     model.HandlerTransport
	}
)

func NewGrpcService(
	cfg *model.Config,
	logger *zap.Logger,
	sd ServiceDiscovery,
	sh *helper.ServersHelper,
	h model.HandlerTransport,
) (*Service, error) {
	s := &Service{
		logger:      logger,
		serviceName: cfg.ServiceName,
		peers:       make(map[string]Messenger),
		clientCode:  uuid.New().String(),
		sd:          sd,
		handler:     h,
	}

	sh.AddDefinition(helper.Definition{
		Description:    &proto.Transport_ServiceDesc,
		Implementation: newServer(s.Process),
	})

	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	go s.startChecking(ctx)

	return nil
}

func (s *Service) Stop(_ context.Context) error {
	for _, p := range s.peers {
		if err := p.Close(); err != nil {
			s.logger.Error("close peers err", zap.Error(err))
		}
	}

	return nil
}

func (s *Service) SendMessage(typeMessage string, payload []byte) error {
	s.muPeers.Lock()
	defer s.muPeers.Unlock()

	for _, p := range s.peers {
		if err := p.Send(&proto.Message{
			Type:        typeMessage,
			MessageCode: uuid.New().String(),
			ClientCode:  s.clientCode,
			Timestamp:   timestamppb.Now(),
			Payload:     payload,
		}); err != nil {
			return fmt.Errorf("send err: %w", err)
		}
	}

	return nil
}

func (s *Service) Process(stream proto.Transport_ProcessServer) error {
	clientCode, err := s.processHandShakeServer(stream)
	if err != nil {
		return fmt.Errorf("handshake err: %w", err)
	}

	return s.processStream(stream, clientCode)
}

func (s *Service) processHandShakeServer(stream proto.Transport_ProcessServer) (string, error) {
	clinetCode, err := s.processHandshake(stream)
	if err != nil {
		return "", fmt.Errorf("handshake err: %w", err)
	}

	if clinetCode == s.clientCode {
		return "", ErrItsMeMario
	}

	s.muPeers.Lock()
	if s.peers[clinetCode] != nil {
		s.muPeers.Unlock()
		return "", ErrServiceExists
	}

	if err = s.sendHandshake(stream); err != nil {
		s.muPeers.Unlock()

		return "", fmt.Errorf("send err: %w", err)
	}

	s.peers[clinetCode] = &MessengerServer{stream: stream}
	s.muPeers.Unlock()

	return clinetCode, nil
}

func (s *Service) processStream(stream Stream, clientCode string) error {
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Error("panic", zap.Any("recover", rec))
		}

		s.muPeers.Lock()
		delete(s.peers, clientCode)
		s.muPeers.Unlock()
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			s.logger.Error("recv err", zap.Error(err))
			return fmt.Errorf("recv err: %w", err)
		}

		if msg.GetType() == proto.ServiceMessageType_SERVICE_MESSAGE_TYPE_HANDSHAKE.String() {
			s.logger.Error("another handshake")
			continue
		}

		if err = s.handler(msg.GetType(), msg.GetMessageCode(), msg.GetPayload()); err != nil {
			s.logger.Error("handler err", zap.Error(err))
			continue
		}
	}
}

func (s *Service) startChecking(ctx context.Context) {
	ticker := time.NewTicker(discoveryTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		instances, err := s.sd.GetServicesByServiceName(s.serviceName)
		if err != nil {
			s.logger.Error("get instances err", zap.Error(err))
			continue
		}

		for _, instance := range instances {
			client, errC := s.getClient(instance)
			if errC != nil {
				s.logger.Error("grpc dial errC", zap.Error(errC))
			}

			stream, errC := client.Process(ctx)
			if errC != nil {
				s.logger.Error("set kv system errC", zap.Error(errC))
				continue
			}

			if errC = s.sendHandshake(stream); errC != nil && !errors.Is(errC, ErrSkipConnect) {
				s.logger.Error("send errC", zap.Error(errC))
				continue
			}

			if errors.Is(errC, ErrSkipConnect) {
				continue
			}

			clientCode, errC := s.clientRecv(stream)
			if errC != nil {
				s.logger.Error("recv errC", zap.Error(errC))
			}

			go func() {
				if err = s.processStream(stream, clientCode); err != nil {
					s.logger.Error("process stream err", zap.Error(err))
				}
			}()
		}
	}
}

func (s *Service) sendHandshake(stream Stream) error {
	if errC := stream.Send(&proto.Message{
		Type:        proto.ServiceMessageType_SERVICE_MESSAGE_TYPE_HANDSHAKE.String(),
		MessageCode: uuid.New().String(),
		ClientCode:  s.clientCode,
		Timestamp:   timestamppb.Now(),
	}); errC != nil {
		st, ok := status.FromError(errC)
		if !ok {
			return fmt.Errorf("send errC: %w", errC)
		}

		if st.Code() == codes.Code(99999) || st.Code() == codes.AlreadyExists {
			return ErrSkipConnect
		}
	}

	return nil
}

func (s *Service) processHandshake(stream Stream) (string, error) {
	msg, errC := stream.Recv()
	if errC == io.EOF {
		return "", fmt.Errorf("eof: %w", errC)
	}

	if errC != nil {
		return "", fmt.Errorf("recv errC: %w", errC)
	}

	if msg.GetType() != proto.ServiceMessageType_SERVICE_MESSAGE_TYPE_HANDSHAKE.String() {
		return "", ErrMissHandshake
	}

	return msg.GetClientCode(), nil
}

func (s *Service) clientRecv(stream proto.Transport_ProcessClient) (string, error) {
	clientCode, err := s.processHandshake(stream)
	if err != nil {
		return "", fmt.Errorf("handshake err: %w", err)
	}

	s.muPeers.Lock()
	if s.peers[clientCode] != nil {
		s.muPeers.Unlock()
		return "", nil
	}

	s.peers[clientCode] = &MessengerClient{stream: stream}
	s.muPeers.Unlock()

	return clientCode, nil
}

func (s *Service) getClient(addr string) (proto.TransportClient, error) {
	d, errC := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			apmgrpc.NewUnaryClientInterceptor(),
		),
	)
	if errC != nil {
		return nil, fmt.Errorf("grpc dial errC: %w", errC)
	}

	return proto.NewTransportClient(d), nil
}
