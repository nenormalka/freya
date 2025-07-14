package redis

import (
	"github.com/nenormalka/freya/conns/redis/config"
	"github.com/nenormalka/melissa/types"
)

var Module = types.Module{
	{CreateFunc: config.CreateConfig},
	{CreateFunc: NewRedisClient},
}
