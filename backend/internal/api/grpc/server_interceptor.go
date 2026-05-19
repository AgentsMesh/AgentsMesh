package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
)

func loggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		logger.Debug("unary RPC call", "method", info.FullMethod)
		return handler(ctx, req)
	}
}

func loggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Debug("stream RPC call", "method", info.FullMethod)
		return handler(srv, ss)
	}
}
