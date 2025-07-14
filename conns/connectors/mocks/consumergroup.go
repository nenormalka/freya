package mocks

import "github.com/nenormalka/freya/conns/kafka/common"

type (
	ConsumerGroupMock struct {
		consumer *ConsumerMock
	}

	ConsumerGroupMockOption func(cgm *ConsumerGroupMock)
)

func WithConsumerGroupMockMessages(messages []KafkaMessagesMock) ConsumerGroupMockOption {
	return func(cm *ConsumerGroupMock) {
		cm.consumer.messages = messages
	}
}

func NewConsumerGroupMock(opts ...ConsumerGroupMockOption) *ConsumerGroupMock {
	cgm := &ConsumerGroupMock{
		consumer: NewConsumerMock(),
	}

	for _, opt := range opts {
		opt(cgm)
	}

	return cgm
}

func (cgm *ConsumerGroupMock) AddHandler(topic common.Topic, hm common.MessageHandler) error {
	return cgm.consumer.AddHandler(topic, hm)
}

func (cgm *ConsumerGroupMock) Consume() error {
	return cgm.consumer.Consume()
}

func (cgm *ConsumerGroupMock) Close() error {
	return cgm.consumer.Close()
}

func (cgm *ConsumerGroupMock) PauseAll() {
	cgm.consumer.PauseAll()
}

func (cgm *ConsumerGroupMock) ResumeAll() {
	cgm.consumer.ResumeAll()
}
