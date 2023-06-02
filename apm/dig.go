package apm

import (
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewAPMConfig},
	{CreateFunc: NewAPM},
}
