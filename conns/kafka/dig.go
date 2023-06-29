package kafka

import (
	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewKafka},
	{CreateFunc: CreateConfig},
}

func CreateConfig(cfg *config.Config) common.Config {
	if len(cfg.Kafka.Addresses) == 0 {
		return common.Config{}
	}

	skipUnmarshal := make(map[common.Topic]struct{})
	for _, addr := range cfg.Kafka.SkipUnmarshal {
		if addr != "" {
			skipUnmarshal[common.Topic(addr)] = struct{}{}
		}
	}

	return common.Config{
		Addresses:           cfg.Kafka.Addresses,
		SkipUnmarshalErrors: skipUnmarshal,
	}
}
