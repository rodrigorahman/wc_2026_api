// Package interceptor provides cross-cutting unary gRPC server interceptors:
// proto message validation (protovalidate) and JWT bearer authentication.
package interceptor

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// NewProtoValidateUnaryInterceptor returns a unary server interceptor that
// validates each request message against its buf.validate rules before the
// handler runs. A validation failure short-circuits the call with
// codes.InvalidArgument and the handler is never invoked.
func NewProtoValidateUnaryInterceptor(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		msg, ok := req.(proto.Message)
		if !ok {
			return handler(ctx, req)
		}

		if err := validator.Validate(msg); err != nil {
			var valErr *protovalidate.ValidationError
			if errors.As(err, &valErr) {
				return nil, status.Error(codes.InvalidArgument, valErr.Error())
			}
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return handler(ctx, req)
	}
}
