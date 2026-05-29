package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary interceptor that logs each RPC
// with the fields: rpc (full method), code (gRPC status code), duration_ms.
//
// Log levels follow §13.1 and §10.3 of the Tech Spec:
//   - codes.OK                          → info
//   - domain errors (4xx-equivalent)    → warn
//   - internal / unexpected errors      → error
//
// No sensitive fields (plain passwords, password_hash, JWT tokens) are ever logged.
func UnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		durationMs := float64(time.Since(start).Microseconds()) / 1000.0
		code := status.Code(err)

		fields := []zap.Field{
			zap.String("rpc", info.FullMethod),
			zap.String("code", code.String()),
			zap.Float64("duration_ms", durationMs),
		}

		switch {
		case err == nil:
			logger.Info("rpc handled", fields...)
		case isDomainError(code):
			logger.Warn("rpc handled", fields...)
		default:
			logger.Error("rpc handled", append(fields, zap.Error(err))...)
		}

		return resp, err
	}
}

// isDomainError reports whether the gRPC status code represents a client-side
// or domain error (4xx-equivalent), as opposed to an internal server error.
func isDomainError(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.FailedPrecondition,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Canceled,
		codes.DeadlineExceeded,
		codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
