package postrgres

import (
	"github.com/nenormalka/freya/types"
)

var Module = types.Module{
	{CreateFunc: NewPostgresConfig},
	{CreateFunc: NewPostgres},
	{CreateFunc: NewSQLConnector},
	{CreateFunc: NewGoQuConnector},
	{CreateFunc: NewPGXPoolConn},
	{CreateFunc: NewPGXPool},
}
