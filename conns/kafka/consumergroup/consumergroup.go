package consumergroup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/chapsuk/wait"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/types"
)

var (
	ErrTopicExists = errors.New("topic already exists")
)

type (
	ConsumerGroup struct {
		name       string
		skipErrors map[common.Topic]struct{}
		topics     common.Topics
		handlers   map[common.Topic]common.MessageHandler
		closed     chan struct{}

		logger  *zap.Logger
		config  *sarama.Config
		errFunc common.ErrFunc
		group   sarama.ConsumerGroup
		wg      *wait.Group
		mu      *sync.RWMutex
		sess    sarama.ConsumerGroupSession
	}

	ConsumerGroupOption func(cg *ConsumerGroup)
)

func ConfigOption(cfg *sarama.Config) ConsumerGroupOption {
	return func(cg *ConsumerGroup) {
		cg.config = cfg
	}
}

func ErrFuncOption(f common.ErrFunc) ConsumerGroupOption {
	return func(cg *ConsumerGroup) {
		cg.errFunc = f
	}
}

func NewConsumerGroup(
	cfg common.Config,
	name string,
	logger *zap.Logger,
	opts ...ConsumerGroupOption,
) (*ConsumerGroup, error) {
	if name == "" {
		return nil, common.ErrEmptyGroupName
	}

	if len(cfg.Addresses) == 0 {
		return nil, common.ErrEmptyAddresses
	}

	metrics.UseNilMetrics = true

	cg := &ConsumerGroup{
		name:       name,
		config:     sarama.NewConfig(),
		skipErrors: cfg.SkipErrors,
		logger:     logger,
		handlers:   make(map[common.Topic]common.MessageHandler),
		closed:     make(chan struct{}),
		wg:         &wait.Group{},
		mu:         &sync.RWMutex{},
		errFunc: func(err error) {
			if err != nil {
				logger.Error(fmt.Sprintf("consume on topic %s", name), zap.Error(err))
			}
		},
	}

	for _, opt := range opts {
		opt(cg)
	}

	if cg.config == nil {
		return nil, common.ErrEmptyConfig
	}

	if cg.errFunc == nil {
		return nil, common.ErrEmptyErrFunc
	}

	var err error
	cg.group, err = sarama.NewConsumerGroup(cfg.Addresses, name, cg.config)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	cg.wg.Add(cg.serveErrors)

	return cg, nil
}

func (cg *ConsumerGroup) AddHandler(topic common.Topic, hm common.MessageHandler) error {
	if _, ok := cg.handlers[topic]; ok {
		return ErrTopicExists
	}

	cg.handlers[topic] = hm
	cg.topics = append(cg.topics, topic)

	return nil
}

func (cg *ConsumerGroup) Consume() error {
	if len(cg.handlers) == 0 {
		return common.ErrEmptyHandlers
	}

	if len(cg.topics) == 0 {
		return common.ErrEmptyTopics
	}

	cg.wg.Add(func() {
		for {
			select {
			case <-cg.closed:
				return
			default:
			}

			if err := cg.group.Consume(context.Background(), cg.topics.ToStrings(), cg); err != nil {
				cg.errFunc(err)
			}
		}
	})

	return nil
}

func (cg *ConsumerGroup) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	cg.logger.Info(fmt.Sprintf("Kafka: start consume topic: %s partition: %d", claim.Topic(), claim.Partition()))

	handler, ok := cg.handlers[common.Topic(claim.Topic())]
	if !ok {
		return fmt.Errorf("missing handler for topic: %s", claim.Topic())
	}

	for msg := range claim.Messages() {
		start := time.Now()
		if err := handler(msg.Value); err != nil {
			cg.errFunc(fmt.Errorf("handle %s topic: err %w", msg.Topic, err))

			types.KafkaConsumerGroupMetricsF(cg.name, msg.Topic, err, time.Since(start).Seconds())

			if _, ok = cg.skipErrors[common.Topic(claim.Topic())]; ok {
				continue
			}

			return fmt.Errorf("ConsumeClaim topic %s, err %w", msg.Topic, err)
		}

		sess.MarkMessage(msg, "ok")

		types.KafkaConsumerGroupMetricsF(cg.name, msg.Topic, nil, time.Since(start).Seconds())
	}

	return nil
}

func (cg *ConsumerGroup) Setup(sess sarama.ConsumerGroupSession) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	cg.sess = sess

	return nil
}

func (cg *ConsumerGroup) Cleanup(_ sarama.ConsumerGroupSession) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	cg.sess = nil

	return nil
}

func (cg *ConsumerGroup) Close() error {
	select {
	case <-cg.closed:
		return common.ErrGroupAlreadyClosed
	default:
	}

	close(cg.closed)
	err := cg.group.Close()
	cg.wg.Wait()

	if err != nil {
		return fmt.Errorf("consumer group %s, close err %w", cg.name, err)
	}

	return nil
}

func (cg *ConsumerGroup) PauseAll() {
	cg.group.PauseAll()
}

func (cg *ConsumerGroup) ResumeAll() {
	cg.group.ResumeAll()
}

func (cg *ConsumerGroup) serveErrors() {
	for err := range cg.group.Errors() {
		cg.errFunc(err)
	}
}
