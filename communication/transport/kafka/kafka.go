package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/nenormalka/freya/communication/model"
	"github.com/nenormalka/freya/communication/proto"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/kafka"
	"github.com/nenormalka/freya/conns/kafka/common"

	"go.uber.org/zap"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrEmptyTopic = errors.New("empty topic")
)

type (
	Service struct {
		logger     *zap.Logger
		handler    model.HandlerTransport
		consumer   kafka.Consumer
		producer   kafka.SyncProducer
		clientCode string
		topic      common.Topic
	}
)

func NewKafkaService(
	cfg *model.Config,
	logger *zap.Logger,
	c *conns.Conns,
	h model.HandlerTransport,
) (*Service, error) {
	if cfg.KafkaTopic == "" {
		return nil, ErrEmptyTopic
	}

	k, err := c.GetKafka()
	if err != nil {
		return nil, fmt.Errorf("c.GetKafka(): %w", err)
	}

	consumer, err := k.NewConsumer()
	if err != nil {
		return nil, fmt.Errorf("NewConsumer(): %w", err)
	}

	producer, err := k.NewSyncProducer()
	if err != nil {
		return nil, fmt.Errorf("k.NewSyncProducer(): %w", err)
	}

	return &Service{
		logger:     logger,
		clientCode: uuid.New().String(),
		topic:      common.Topic(cfg.KafkaTopic),
		consumer:   consumer,
		producer:   producer,
		handler:    h,
	}, nil
}

func (s *Service) Start(_ context.Context) error {
	if err := s.setTopicHandler(); err != nil {
		return fmt.Errorf("s.setTopicHandler(): %w", err)
	}

	if err := s.consumer.Consume(); err != nil {
		return fmt.Errorf("s.consumer.Consume() %w", err)
	}

	return nil
}

func (s *Service) Stop(_ context.Context) error {
	if err := s.consumer.Close(); err != nil {
		return fmt.Errorf("s.consumer.Close() %w", err)
	}

	if err := s.producer.Close(); err != nil {
		return fmt.Errorf("s.producer.Close(): %w", err)
	}

	return nil
}

func (s *Service) SendMessage(typeMessage string, payload []byte) error {
	if err := kafka.TypedSend(s.producer, s.topic, &proto.Message{
		Type:        typeMessage,
		MessageCode: uuid.New().String(),
		ClientCode:  s.clientCode,
		Timestamp:   timestamppb.Now(),
		Payload:     payload,
	}); err != nil {
		return fmt.Errorf("kafka.TypedSend(): %w", err)
	}

	return nil
}

func (s *Service) setTopicHandler() error {
	if err := kafka.AddTypedHandlerConsumer(s.consumer, s.topic, func(msg *proto.Message) error {
		if err := s.handler(msg.GetType(), msg.GetMessageCode(), msg.GetPayload()); err != nil && !errors.Is(err, model.ErrInvalidMessageType) {
			return fmt.Errorf("s.handler(): %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("AddTypedHandlerConsumer(): %w", err)
	}

	return nil
}
