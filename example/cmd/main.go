package main

import (
	"freya/example"
	exampleconfig "freya/example/config"
	grpc "freya/example/grpc"
	"freya/example/http"
	"freya/example/outbox"
	"freya/example/repo"
	"freya/example/service"

	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/types"
	melissa "github.com/nenormalka/melissa/types"
)

var releaseID = "release-id-example"

var Module = melissa.Module{
	{CreateFunc: func() (*types.AppInfo, error) {
		return types.GetAppInfo(example.ModInfo, releaseID, "")
	}},
	{CreateFunc: repo.NewRepo},
}.
	Append(exampleconfig.Module).
	Append(grpc.Module).
	Append(service.Module).
	Append(http.Module).
	Append(outbox.Module)

func main() {
	freya.
		NewEngine(Module).
		Run()
}
