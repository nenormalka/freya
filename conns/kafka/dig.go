package kafka

import (
	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns/kafka/common"
	"github.com/nenormalka/freya/types"
	"strings"
)

var Module = types.Module{
	{CreateFunc: NewKafka},
	{CreateFunc: CreateConfig},
}

func CreateConfig(cfg *config.Config) common.Config {
	if cfg.Kafka.Addresses == "" {
		return common.Config{}
	}

	addrs := strings.Split(cfg.Kafka.Addresses, ",")

	skipErrors := make(map[common.Topic]struct{})
	for _, topic := range strings.Split(cfg.Kafka.SkipErrors, ",") {
		if topic != "" {
			skipErrors[common.Topic(topic)] = struct{}{}
		}
	}

	return common.Config{
		Addresses:   addrs,
		SkipErrors:  skipErrors,
		EnableDebug: cfg.Kafka.EnableDebug,
		AppName:     cfg.AppName,
	}
}
