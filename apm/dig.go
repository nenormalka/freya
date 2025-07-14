package apm

import "github.com/nenormalka/melissa/types"

var Module = types.Module{
	{CreateFunc: NewAPMConfig},
	{CreateFunc: NewAPM},
}
