package metadata

import (
	"context"

	"github.com/google/uuid"
	lilith "github.com/nenormalka/lilith/methods"
	"google.golang.org/grpc/metadata"
)

const (
	FreyaTraceID     = "freya_trace_id"
	FreyaMethodsPath = "freya_methods_path"
)

func GetFreyaTraceID(ctx context.Context) (context.Context, string) {
	id, err := GetDataFromCtx(ctx, FreyaTraceID)
	if err == nil {
		return ctx, id
	}

	id = uuid.NewString()

	return metadata.AppendToOutgoingContext(ctx, FreyaTraceID, id), id
}

func GetFreyaMethodsPath(ctx context.Context, method string) (context.Context, string) {
	path, _ := GetDataFromCtx(ctx, FreyaMethodsPath)
	path = lilith.Ternary(path == "", method, path+"->"+method)

	return metadata.AppendToOutgoingContext(ctx, FreyaMethodsPath, path), path
}
