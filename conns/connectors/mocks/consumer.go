package mocks

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/conns/kafka/consumer"
)

type (
	ConsumerMock struct {
		enabled  bool
		closeCh  chan struct{}
		mu       *sync.Mutex
		messages []KafkaMessagesMock
		handlers map[common.Topic]common.MessageHandler
		channels map[common.Topic]chan json.RawMessage
	}

	KafkaMessagesMock struct {
		Message []json.RawMessage
		Topic   common.Topic
		Tick    time.Duration
		Cycle   bool
	}

	ConsumerMockOption func(cm *ConsumerMock)
)

func WithConsumerMockMessages(messages []KafkaMessagesMock) ConsumerMockOption {
	return func(cm *ConsumerMock) {
		cm.messages = messages
	}
}

func NewConsumerMock(opts ...ConsumerMockOption) *ConsumerMock {
	cm := &ConsumerMock{
		enabled:  false,
		closeCh:  make(chan struct{}),
		mu:       &sync.Mutex{},
		messages: []KafkaMessagesMock{},
		handlers: make(map[common.Topic]common.MessageHandler),
		channels: make(map[common.Topic]chan json.RawMessage),
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

func (cm *ConsumerMock) Consume() error {
	if len(cm.messages) == 0 || len(cm.handlers) == 0 {
		return nil
	}

	cm.enabled = true
	cm.startMessaging()
	cm.startHandling()

	return nil
}

func (cm *ConsumerMock) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	close(cm.closeCh)

	for _, ch := range cm.channels {
		close(ch)
	}

	return nil
}

func (cm *ConsumerMock) PauseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.enabled = false
}

func (cm *ConsumerMock) ResumeAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.enabled = true
}

func (cm *ConsumerMock) AddHandler(topic common.Topic, mh common.MessageHandler, opts ...consumer.HandlerOption) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.handlers[topic] = mh

	return nil
}

func (cm *ConsumerMock) startHandling() {
	for topic, ch := range cm.channels {
		mh, ok := cm.handlers[topic]
		if !ok {
			continue
		}

		go func(ch chan json.RawMessage, mh common.MessageHandler) {
			for {
				select {
				case <-cm.closeCh:
					return
				case msg, ok := <-ch:
					if !ok {
						return
					}

					mh(msg)
				}
			}
		}(ch, mh)
	}
}

func (cm *ConsumerMock) startMessaging() {
	for _, messages := range cm.messages {
		if _, ok := cm.handlers[messages.Topic]; !ok {
			continue
		}

		if _, ok := cm.channels[messages.Topic]; !ok {
			cm.channels[messages.Topic] = make(chan json.RawMessage)
		}

		go func(messages KafkaMessagesMock) {
			index := 0

			ticker := time.NewTicker(messages.Tick)
			defer ticker.Stop()

			for {
				select {
				case <-cm.closeCh:
					return
				case <-ticker.C:
					if !cm.enabled {
						continue
					}

					cm.mu.Lock()
					ch, ok := cm.channels[messages.Topic]
					if !ok {
						cm.mu.Unlock()
						return
					}

					ch <- messages.Message[index]

					cm.mu.Unlock()

					index++

					indexOver := index >= len(messages.Message)

					if !messages.Cycle && indexOver {
						return
					}

					if indexOver {
						index = 0
					}

				}
			}
		}(messages)
	}
}
