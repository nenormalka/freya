package postrgres

import "github.com/nenormalka/melissa/types"

var Module = types.Module{
	{CreateFunc: NewPostgresConfig},
	{CreateFunc: NewSQLX},
	{CreateFunc: NewSQLConnector},
	{CreateFunc: NewGoQuConnector},
	{CreateFunc: NewPGXPoolConn},
	{CreateFunc: NewPGXPool},
}
