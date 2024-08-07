package grpc

import (
	"context"
	"fmt"
	"path"
	"runtime/debug"
	"strings"

	"github.com/nenormalka/freya/types"
	"github.com/nenormalka/freya/types/errors"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	sentry "github.com/johnbellone/grpc-middleware-sentry"
	"github.com/nenormalka/bishamon"
	"go.elastic.co/apm/module/apmgrpc/v2"
	"go.elastic.co/apm/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

const (
	fieldName = "grpc.details"
)

func interceptors(
	logger *zap.Logger,
	tracer *apm.Tracer,
	customInterceptors [][]grpc.UnaryServerInterceptor,
	config *Config,
) []grpc.UnaryServerInterceptor {
	ints := []grpc.UnaryServerInterceptor{
		apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery(), apmgrpc.WithTracer(tracer)),
		grpcctxtags.UnaryServerInterceptor(grpcctxtags.WithFieldExtractor(grpcctxtags.CodeGenRequestFieldExtractor)),
		payloadLoggingInterceptor(logger, config),
		logMetadataInterceptor(logger, config),
		initSentryInterceptor([]codes.Code{
			codes.Unknown,
			codes.DeadlineExceeded,
			codes.Internal,
			codes.Unimplemented,
		}),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(panicInterceptor(logger))),
		checkErrorInterceptor(),
	}

	if config.WithServerMetrics {
		ints = append(ints, types.ServerGRPCMetrics.UnaryServerInterceptor())
	}

	for _, customInts := range customInterceptors {
		ints = append(ints, customInts...)
	}

	return ints
}

func panicInterceptor(logger *zap.Logger) func(ctx context.Context, p any) (err error) {
	return func(ctx context.Context, p any) (err error) {
		types.GRPCPanicInc()

		logger.Error(
			"recovered panic",
			zap.String("panic value", fmt.Sprintf("%v", p)),
			zap.ByteString("stacktrace", debug.Stack()),
			fieldWithTraceID(ctx),
		)

		return status.Errorf(codes.Internal, "%s", p)
	}
}

func checkErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (
		resp any, err error,
	) {
		resp, err = handler(ctx, req)
		if err != nil {
			err = errors.ErrorToGRPCError(err)
		}

		return resp, err
	}
}

func logMetadataInterceptor(logger *zap.Logger, config *Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (
		resp any, err error,
	) {
		md, ok := metadata.FromIncomingContext(ctx)

		if !config.WithDebugLog || !ok {
			return handler(ctx, req)
		}

		fields := make([]zap.Field, 0, len(md)+1)
		for key, values := range md {
			value := ""
			if len(values) != 0 {
				value = values[0]
			}

			fields = append(fields, zap.String(key, value))
		}

		fields = append(fields, fieldWithTraceID(ctx))

		logger.Info("metadata", fields...)

		return handler(ctx, req)
	}
}

func marshalPayload(msg any, redactor *bishamon.Redactor) (result string) {
	p, ok := msg.(proto.Message)
	if !ok {
		return "msg is not proto.Message"
	}

	redactedMessage := proto.Clone(p)
	if redactor != nil {
		if err := redactor.Redact(redactedMessage); err != nil {
			return "redact: " + err.Error()
		}
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(redactedMessage)
	if err != nil {
		return "marshal: " + err.Error()
	}

	return string(bytes)
}

func payloadLoggingInterceptor(logger *zap.Logger, config *Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (
		resp any, err error,
	) {
		apiLogger := logger.Named("api")
		methodFld := zap.String("grpc.method", path.Base(info.FullMethod))

		apiLogger.Info(
			fmt.Sprintf("unary call %s", info.FullMethod),
			methodFld,
			zap.String("grpc.payload", marshalPayload(req, config.LogRedactor)),
			fieldWithTraceID(ctx),
		)

		resp, err = handler(ctx, req)

		code := status.Code(err)
		level := grpczap.DefaultCodeToLevel(code)

		responseField := zap.Skip()

		if config.WithDebugLog || zap.WarnLevel.Enabled(level) {
			responseField = zap.String("grpc.payload", marshalPayload(resp, config.LogRedactor))
		}

		apiLogger.Log(
			level,
			fmt.Sprintf("finished unary call with code %s", code.String()),
			methodFld,
			zap.String("grpc.code", code.String()),
			responseField,
			fieldWithGRPCDetailsError(err),
			zap.Error(err),
			fieldWithTraceID(ctx),
		)

		return
	}
}

func initSentryInterceptor(codesToReport []codes.Code) grpc.UnaryServerInterceptor {
	return sentry.UnaryServerInterceptor(
		sentry.WithReportOn(func(err error) bool {
			currentCode := status.Code(err)
			for _, codeToReport := range codesToReport {
				if currentCode == codeToReport {
					return true
				}
			}

			return false
		}),
		sentry.WithRepanicOption(true),
	)
}

func fieldWithTraceID(ctx context.Context) zap.Field {
	trCtx := apm.TransactionFromContext(ctx).TraceContext()
	if trCtx.Trace.Validate() == nil {
		return zap.String("trace.id", trCtx.Trace.String())
	}
	return zap.Skip()
}

func fieldWithGRPCDetailsError(err error) zap.Field {
	st, ok := status.FromError(err)
	if !ok {
		return zap.Skip()
	}
	details := st.Details()
	if len(details) == 0 {
		return zap.String(fieldName, "")
	}

	messages := make([]string, 0, len(details))
	for i := range details {
		m, ok := details[i].(proto.Message)
		if !ok {
			continue
		}

		messages = append(messages, prototext.Format(m))
	}

	return zap.String(fieldName, strings.Join(messages, "\n"))
}
