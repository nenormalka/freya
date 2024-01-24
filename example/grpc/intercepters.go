package example_service

import (
	"context"
	"fmt"
	"time"

	"github.com/nenormalka/freya/metadata"
	"google.golang.org/grpc"
)

func getInterceptors() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		getCustomInterceptor(),
		printMetadataInterceptor(),
	}
}

func printMetadataInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		res := ""

		res, err := metadata.GetAppVersion(ctx)
		if err != nil {
			res = err.Error()
		}
		fmt.Println("md app version", res)

		res, err = metadata.GetPlatform(ctx)
		if err != nil {
			res = err.Error()
		}
		fmt.Println("md platform", res)

		return handler(ctx, req)
	}
}

func getCustomInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		fmt.Println(">>> ", time.Now().String())
		resp, err := handler(ctx, req)
		fmt.Println("<<< ", time.Now().String())
		return resp, err
	}
}
