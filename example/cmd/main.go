package main

import (
	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/types"

	"freya/example"
	exampleconfig "freya/example/config"
	grpc "freya/example/grpc"
	"freya/example/http"
	"freya/example/repo"
	"freya/example/service"
)

var releaseID = "release-id-example"

var Module = types.Module{
	{CreateFunc: func() (*types.AppInfo, error) {
		return types.GetAppInfo(example.ModInfo, releaseID, "")
	}},
	{CreateFunc: repo.NewRepo},
}.
	Append(exampleconfig.Module).
	Append(grpc.Module).
	Append(service.Module).
	Append(http.Module)

func main() {
	freya.
		NewEngine(Module).
		Run()
}
