package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nenormalka/freya/communication/model"
	communicationgrpc "github.com/nenormalka/freya/communication/transport/grpc"
	"github.com/nenormalka/freya/communication/transport/kafka"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/melissa/types"

	"go.uber.org/zap"
)

type (
	Communicator interface {
		SetHandlers(hs map[string]model.HandlerFunc) error
		Transport
	}

	Transport interface {
		SendMessage(typeMessage string, payload []byte) error
		types.Runnable
	}

	Communication struct {
		logger    *zap.Logger
		transport Transport
		handlers  map[string]model.HandlerFunc
	}

	Params struct {
		Connections   *conns.Conns
		ServersHelper *grpc.ServersHelper
	}

	TransportType string
)

func NewCommunication(
	cfg *model.Config,
	logger *zap.Logger,
	params *Params,
) (Communicator, error) {
	if !cfg.Enabled {
		return NewCommunicationMock(), nil
	}

	c := &Communication{
		logger:   logger,
		handlers: make(map[string]model.HandlerFunc),
	}

	if err := c.setTransport(cfg, params); err != nil {
		return nil, fmt.Errorf("create transport err: %w", err)
	}

	return c, nil
}

func (c *Communication) Start(ctx context.Context) error {
	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("start transport err: %w", err)
	}

	return nil

}

func (c *Communication) Stop(ctx context.Context) error {
	if err := c.transport.Stop(ctx); err != nil {
		return fmt.Errorf("stop transport err: %w", err)
	}

	return nil
}

func (c *Communication) SetHandlers(hs map[string]model.HandlerFunc) error {
	c.handlers = hs
	return nil
}

func (c *Communication) SendMessage(typeMessage string, payload []byte) error {
	return c.transport.SendMessage(typeMessage, payload)
}

func (c *Communication) handler(messageType, messageCode string, payload json.RawMessage) error {
	if len(c.handlers) == 0 {
		c.logger.Error("empty handlers")

		return model.ErrEmptyHandlers
	}

	h, ok := c.handlers[messageType]
	if !ok {
		c.logger.Error("handler not found", zap.String("messageType", messageType))

		return model.ErrInvalidMessageType
	}

	return h(messageCode, payload)
}

func (c *Communication) setTransport(cfg *model.Config, params *Params) error {
	var (
		transport Transport
		err       error
	)

	switch cfg.TransportType {
	case model.GRPCTransport:
		sd, errC := c.getGRPCTransportDiscovery(cfg, params)
		if errC != nil {
			return fmt.Errorf("get grpc transport discovery err: %w", errC)
		}

		transport, err = communicationgrpc.NewGrpcService(
			cfg,
			c.logger,
			sd,
			params.ServersHelper,
			c.handler,
		)
	case model.KafkaTransport:
		transport, err = kafka.NewKafkaService(
			cfg,
			c.logger,
			params.Connections,
			c.handler,
		)
	default:
		return fmt.Errorf("unsupported transport: %s", cfg.TransportType)
	}
	if err != nil {
		return fmt.Errorf("create transport err: %w", err)
	}

	c.transport = transport

	return nil
}

func (c *Communication) getGRPCTransportDiscovery(cfg *model.Config, params *Params) (communicationgrpc.ServiceDiscovery, error) {
	if cfg.GRPCAddresses == "" {
		consul, errC := params.Connections.GetConsul()
		if errC != nil {
			return nil, fmt.Errorf("get consul err: %w", errC)
		}

		return consul.ServiceDiscovery(), nil
	}

	return communicationgrpc.NewStaticServiceDiscovery(cfg.GRPCAddresses)
}
