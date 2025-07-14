package sentry

import "github.com/nenormalka/melissa/types"

var Module = types.Module{
	{CreateFunc: NewSentryConfig},
	{CreateFunc: NewSentry},
}
