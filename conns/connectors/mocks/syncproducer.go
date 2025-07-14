package mocks

import (
	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/conns/kafka/syncproducer"
)

type (
	SyncProducerMock struct{}

	SyncProducerMockOption func(spm *SyncProducerMock)
)

func NewSyncProducerMock(opts ...SyncProducerMockOption) *SyncProducerMock {
	return &SyncProducerMock{}
}

func (spm *SyncProducerMock) Send(topic common.Topic, message []byte, opts ...syncproducer.SendOptions) error {
	return nil
}

func (spm *SyncProducerMock) Close() error {
	return nil
}
