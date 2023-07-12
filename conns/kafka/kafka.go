package kafka

import (
	"fmt"
	"github.com/Shopify/sarama"
	"log"
	"os"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/conns/kafka/consumergroup"
	"github.com/nenormalka/freya/conns/kafka/syncproducer"

	"go.uber.org/zap"
)

type (
	ConsumerGroup interface {
		AddHandler(topic common.Topic, hm common.MessageHandler)
		Consume()
		Close() error
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
