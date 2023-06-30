package kafka

import (
	"fmt"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/conns/kafka/consumergroup"

	"go.uber.org/zap"
)

type (
	ConsumerGroup interface {
		AddHandler(topic common.Topic, hm common.MessageHandler)
		Consume()
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
