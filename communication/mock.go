package communication

import (
	"context"

	"github.com/nenormalka/freya/communication/model"
)

type (
	CommunicationMock struct{}
)

func NewCommunicationMock() *CommunicationMock {
	return &CommunicationMock{}
}

func (*CommunicationMock) Start(_ context.Context) error {
	return nil
}

func (*CommunicationMock) Stop(_ context.Context) error {
	return nil
}

func (*CommunicationMock) SetHandlers(_ map[string]model.HandlerFunc) error {
	return nil
}

func (*CommunicationMock) SendMessage(_ string, _ []byte) error {
	return nil
}
