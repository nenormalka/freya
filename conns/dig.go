package conns

import (
	"github.com/nenormalka/freya/conns/elastic"
	"github.com/nenormalka/freya/conns/kafka"
	postrgres "github.com/nenormalka/freya/conns/postgres"
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewConns},
}.
	Append(postrgres.Module).
	Append(elastic.Module).
	Append(kafka.Module)
