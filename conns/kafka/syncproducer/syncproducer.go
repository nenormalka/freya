package syncproducer

import (
	"fmt"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/types"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type (
	SyncProducer struct {
		logger *zap.Logger
		pr     sarama.SyncProducer
		config *sarama.Config
	}

	SyncProducerOption func(sp *SyncProducer)

	SendOptions func(msg *sarama.ProducerMessage)
)

func ConfigOption(cfg *sarama.Config) SyncProducerOption {
	return func(sp *SyncProducer) {
		sp.config = cfg
	}
}

func PartitionKeyOption(partitionKey string) SendOptions {
	return func(msg *sarama.ProducerMessage) {
		msg.Key = sarama.StringEncoder(partitionKey)
	}
}

func HeadersOption(headers []sarama.RecordHeader) SendOptions {
	return func(msg *sarama.ProducerMessage) {
		msg.Headers = headers
	}
}

func MetadataOption(data any) SendOptions {
	return func(msg *sarama.ProducerMessage) {
		msg.Metadata = data
	}
}

func NewSyncProducer(
	cfg common.Config,
	logger *zap.Logger,
	opts ...SyncProducerOption,
) (*SyncProducer, error) {
	if len(cfg.Addresses) == 0 {
		return nil, common.ErrEmptyAddresses
	}

	sp := &SyncProducer{
		logger: logger,
		config: sarama.NewConfig(),
	}

	for _, opt := range opts {
		opt(sp)
	}

	if sp.config == nil {
		return nil, common.ErrEmptyConfig
	}

	sp.config.Producer.Retry.Max = 5
	sp.config.Producer.RequiredAcks = sarama.WaitForAll
	sp.config.Producer.Return.Successes = true

	var err error
	sp.pr, err = sarama.NewSyncProducer(cfg.Addresses, sp.config)
	if err != nil {
		return nil, fmt.Errorf("kafka sync producer err: %w", err)
	}

	return sp, nil
}

func (sp *SyncProducer) Send(topic string, message []byte, opts ...SendOptions) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	for _, opt := range opts {
		opt(msg)
	}

	_, _, err := sp.pr.SendMessage(msg)

	types.KafkaSyncProducerMetricsF(topic, err)

	if err != nil {
		return fmt.Errorf("send message err: %w", err)
	}

	return nil
}

func (sp *SyncProducer) Close() error {
	if err := sp.pr.Close(); err != nil {
		return fmt.Errorf("kafka sync producer close err: %w", err)
	}

	return nil
}
