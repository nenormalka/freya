package common

import "encoding/json"

type (
	MessageHandler func(msg json.RawMessage) error

	ErrFunc func(err error)

	Topic string

	Topics []Topic
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
