package model

import (
	"encoding/json"
	"errors"
)

const (
	GRPCTransport  = "grpc"
	KafkaTransport = "kafka"
)

var (
	ErrEmptyHandlers      = errors.New("empty handlers")
	ErrInvalidMessageType = errors.New("invalid message type")
)

type (
	HandlerFunc      func(messageCode string, h json.RawMessage) error
	HandlerTransport func(messageType, messageCode string, payload json.RawMessage) error
)
