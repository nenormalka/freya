package main

import (
	grpc "github.com/nenormalka/freya/example/grpc"
	"github.com/nenormalka/freya/example/http"
	"github.com/nenormalka/freya/example/repo"
	"github.com/nenormalka/freya/example/service"

	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/types"
)

var releaseID = "release-id-example"

var Module = types.Module{
	{CreateFunc: func() config.ReleaseID {
		return config.ReleaseID(releaseID)
	}},
	{CreateFunc: repo.NewRepo},
}.
	Append(grpc.Module).
	Append(service.Module).
	Append(http.Module)

func main() {
	freya.
		NewEngine(Module).
		Run()
}
