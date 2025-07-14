package elastic

import "github.com/nenormalka/melissa/types"

var Module = types.Module{
	{CreateFunc: NewElastic},
	{CreateFunc: NewConfig},
	{CreateFunc: NewElasticConn},
}
