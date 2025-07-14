package consumer

import (
	"fmt"
	"time"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/types"

	"github.com/IBM/sarama"
	"github.com/chapsuk/wait"

	"go.uber.org/zap"
)

type (
	ConsumerOption func(c *Consumer)

	HandlerOption func(h *handlerWithOffset)

	Consumer struct {
		config         *sarama.Config
		errCh          func(err error)
		client         sarama.Client
		consumer       sarama.Consumer
		partsConsumers []sarama.PartitionConsumer
		handlers       map[common.Topic]*handlerWithOffset
		wg             wait.Group
	}

	handlerWithOffset struct {
		timeOffset time.Time
		handler    common.MessageHandler
	}
)

func ConfigOption(cfg *sarama.Config) ConsumerOption {
	return func(c *Consumer) {
		c.config = cfg
	}
}

func ErrorHandlerOption(f func(err error)) ConsumerOption {
	return func(c *Consumer) {
		c.errCh = f
	}
}

func TimeOffsetOption(timeOffset time.Time) HandlerOption {
	return func(t *handlerWithOffset) {
		t.timeOffset = timeOffset
	}
}

func NewConsumer(
	cfg common.Config,
	logger *zap.Logger,
	opts ...ConsumerOption,
) (*Consumer, error) {
	c := &Consumer{
		config:   sarama.NewConfig(),
		errCh:    func(err error) { logger.Error("kafka consumer error", zap.Error(err)) },
		handlers: make(map[common.Topic]*handlerWithOffset),
		wg:       wait.Group{},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.config == nil {
		return nil, common.ErrEmptyConfig
	}

	if c.errCh == nil {
		return nil, common.ErrEmptyErrFunc
	}

	var err error
	c.client, err = sarama.NewClient(cfg.Addresses, c.config)
	if err != nil {
		return nil, fmt.Errorf("sarama.NewClient: %w", err)
	}

	c.consumer, err = sarama.NewConsumerFromClient(c.client)
	if err != nil {
		return nil, fmt.Errorf("sarama.NewConsumerFromClient: %w", err)
	}

	return c, nil
}

func (c *Consumer) Consume() error {
	if len(c.handlers) == 0 {
		return common.ErrEmptyHandlers
	}

	for topic, th := range c.handlers {
		if err := c.consumeTopic(topic, th); err != nil {
			return fmt.Errorf("c.consumeTopic err topic %s timeOffset %s: %w", topic, th.timeOffset, err)
		}
	}

	return nil
}

func (c *Consumer) Close() error {
	for _, pc := range c.partsConsumers {
		if err := pc.Close(); err != nil {
			c.errCh(fmt.Errorf("pc.Close: %w", err))
		}
	}

	c.wg.Wait()

	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("consumer.Close: %w", err)
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("client.Close: %w", err)
	}

	return nil
}

func (c *Consumer) AddHandler(topic common.Topic, mh common.MessageHandler, opts ...HandlerOption) error {
	if _, ok := c.handlers[topic]; ok {
		return common.ErrTopicExists
	}

	tc := &handlerWithOffset{
		timeOffset: time.Time{},
		handler:    mh,
	}

	for _, opt := range opts {
		opt(tc)
	}

	c.handlers[topic] = tc

	return nil
}

func (c *Consumer) PauseAll() {
	c.consumer.PauseAll()
}

func (c *Consumer) ResumeAll() {
	c.consumer.ResumeAll()
}

func (c *Consumer) consumeTopic(topic common.Topic, h *handlerWithOffset) error {
	parts, err := c.getTopicPartitions(topic)
	if err != nil {
		return fmt.Errorf("getTopicPartitions: %w", err)
	}

	for _, part := range parts {
		offset, errOffset := c.getPartitionsOffset(topic, h, part)
		if errOffset != nil {
			return fmt.Errorf("getPartitionsOffset: %w", errOffset)
		}

		pc, errConsume := c.consumer.ConsumePartition(topic.String(), part, offset)
		if errConsume != nil {
			return fmt.Errorf("consumer.ConsumePartition topic %s partition %d offset %d: %w",
				topic.String(), part, offset, errConsume)
		}

		c.consumePartition(topic.String(), part, pc, h)
	}

	return nil
}

func (c *Consumer) getPartitionsOffset(
	topic common.Topic,
	h *handlerWithOffset,
	part int32,
) (int64, error) {
	if h.timeOffset.IsZero() {
		return c.config.Consumer.Offsets.Initial, nil
	}

	offset, err := c.client.GetOffset(topic.String(), part, h.timeOffset.UnixNano()/int64(time.Millisecond))
	if err != nil {
		return offset, fmt.Errorf("client.GetOffset topic %s partition %d time: %s offset: %w",
			topic.String(), part, h.timeOffset, err)
	}

	if offset == -1 {
		offset = sarama.OffsetNewest
	}

	return offset, nil
}

func (c *Consumer) getTopicPartitions(topic common.Topic) ([]int32, error) {
	parts, err := c.client.Partitions(topic.String())
	if err != nil {
		return nil, fmt.Errorf("client.Partitions topic %s: %w", topic, err)
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: %s", common.ErrEmptyPartitions, topic)
	}

	return parts, nil
}

func (c *Consumer) consumePartition(
	topic string,
	part int32,
	pc sarama.PartitionConsumer,
	h *handlerWithOffset,
) {
	c.partsConsumers = append(c.partsConsumers, pc)

	c.wg.Add(func() {
		for msg := range pc.Messages() {
			if !h.timeOffset.IsZero() && msg.Timestamp.Unix() < h.timeOffset.Unix() {
				continue
			}

			start := time.Now()

			err := h.handler(msg.Value)
			if err != nil {
				c.errCh(fmt.Errorf("handler topic %s partition %d : %w", topic, part, err))
			}

			types.KafkaConsumerMetricsF(topic, err, time.Since(start).Seconds())
		}
	})

	if c.config.Consumer.Return.Errors {
		c.wg.Add(func() {
			for err := range pc.Errors() {
				c.errCh(fmt.Errorf("pc.Errors topic %s partition %d : %w", topic, part, err))
			}
		})
	}
}
