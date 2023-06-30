package common

import (
	"encoding/json"
	"errors"
)

type (
	MessageHandler func(msg json.RawMessage) error

	ErrFunc func(err error)

	Topic string

	Topics []Topic
)

var (
	ErrEmptyConfig        = errors.New("err empty config")
	ErrEmptyAddresses     = errors.New("err empty addresses")
	ErrEmptyErrFunc       = errors.New("err empty err func")
	ErrEmptyGroupName     = errors.New("err empty group name")
	ErrGroupAlreadyClosed = errors.New("err group already closed")
)

func (t Topics) ToStrings() []string {
	if len(t) == 0 {
		return nil
	}

	res := make([]string, len(t))
	for i := range t {
		res[i] = string(t[i])
	}

	return res
}
