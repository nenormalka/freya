package couchbase

import "github.com/nenormalka/freya/types"

var Module = types.Module{
	{CreateFunc: NewConfig},
	{CreateFunc: NewCouchbase},
}
