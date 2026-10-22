package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx, end := Start(ctx, "grpc.server", attribute.String("rpc_method", info.FullMethod))
	resp, err := handler(ctx, req)
	code := status.Code(err)
	outcome := code.String()
	if code == codes.OK {
		outcome = "success"
	}
	end(outcome)
	slog.InfoContext(ctx, "rpc request finished", "method", info.FullMethod, "code", code.String())
	return resp, err
}
