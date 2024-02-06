package consul

import (
	"github.com/nenormalka/freya/conns/consul/config"
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewConsul},
	{CreateFunc: config.CreateConfig},
}
