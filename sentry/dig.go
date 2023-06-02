package sentry

import (
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewSentryConfig},
	{CreateFunc: NewSentry},
}
