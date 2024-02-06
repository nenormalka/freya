package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/conns/kafka/consumergroup"
	"github.com/nenormalka/freya/conns/kafka/syncproducer"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type (
	ConsumerGroup interface {
		AddHandler(topic common.Topic, hm common.MessageHandler) error
		Consume() error
		Close() error
		PauseAll()
		ResumeAll()
	}

	SyncProducer interface {
		Send(topic string, message []byte, opts ...syncproducer.SendOptions) error
		Close() error
	}

	Kafka struct {
		cfg    common.Config
		logger *zap.Logger
	}
)

func NewKafka(cfg common.Config, logger *zap.Logger) *Kafka {
	if len(cfg.Addresses) == 0 {
		return nil
	}

	if cfg.EnableDebug {
		sarama.Logger = log.New(os.Stdout, fmt.Sprintf("[%s - KAFKA] - ", cfg.AppName), log.LstdFlags)
	}

	return &Kafka{
		cfg:    cfg,
		logger: logger,
	}
}

func (k *Kafka) NewConsumerGroup(nameGroup string, opts ...consumergroup.ConsumerGroupOption) (ConsumerGroup, error) {
	gr, err := consumergroup.NewConsumerGroup(k.cfg, nameGroup, k.logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka: create consumer group err: %w", err)
	}

	return gr, nil
}

func (k *Kafka) NewSyncProducer(opts ...syncproducer.SyncProducerOption) (SyncProducer, error) {
	sp, err := syncproducer.NewSyncProducer(k.cfg, k.logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka: create sync producer err: %w", err)
	}

	return sp, nil
}

func AddTypedHandler[T any](
	cg ConsumerGroup,
	topic common.Topic,
	f common.MessageHandlerTyped[T],
) error {
	if cg == nil {
		return common.ErrEmptyConsumerGroup
	}

	if err := cg.AddHandler(topic, func(msg json.RawMessage) error {
		var t T

		if err := json.Unmarshal(msg, &t); err != nil {
			return fmt.Errorf("unmarshal message from topic %s err: %w", topic, err)
		}

		return f(t)
	}); err != nil {
		return fmt.Errorf("add handler to topic %s err: %w", topic, err)
	}

	return nil
}

func TypedSend[T any](
	sp SyncProducer,
	topic string,
	message T,
	opts ...syncproducer.SendOptions,
) error {
	if sp == nil {
		return common.ErrEmptySyncProducer
	}

	msg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message to topic %s err: %w", topic, err)
	}

	return sp.Send(topic, msg, opts...)
}
