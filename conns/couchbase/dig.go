package couchbase

import "github.com/nenormalka/melissa/types"

var Module = types.Module{
	{CreateFunc: NewConfig},
	{CreateFunc: NewCouchbase},
}
