package elastic

import "github.com/nenormalka/freya/types"

var Module = types.Module{
	{CreateFunc: NewElastic},
	{CreateFunc: NewConfig},
}
