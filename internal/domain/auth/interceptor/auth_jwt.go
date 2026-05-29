package interceptor

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// bearerScheme is the authorization scheme expected in the metadata.
const bearerScheme = "bearer"

// TokenManager validates a signed access token and returns its sub claim.
// Declared in the consumer package (this interceptor) so it can be mocked or
// substituted; the production implementation lives in internal/auth/token.
type TokenManager interface {
	Validate(tokenString string) (string, error)
}

// ctxKey is an unexported type for context keys defined in this package,
// preventing collisions with keys from other packages.
type ctxKey struct{}

// subKey is the context key under which the authenticated user id (sub claim)
// is stored.
var subKey = ctxKey{}

// SubjectFromContext returns the authenticated user id injected by the auth
// interceptor, and whether it was present.
func SubjectFromContext(ctx context.Context) (string, bool) {
	sub, ok := ctx.Value(subKey).(string)
	return sub, ok
}

// NewAuthJWTUnaryInterceptor returns a unary server interceptor that enforces a
// valid `authorization: Bearer <jwt>` on protected RPCs. The token is validated
// via tm (signature, expiration and algorithm); on success the sub claim is
// injected into the context. Methods in publicMethods (matched by FullMethod)
// bypass authentication. A missing, malformed, expired or otherwise invalid
// token yields codes.Unauthenticated and the handler is never invoked.
func NewAuthJWTUnaryInterceptor(tm TokenManager, publicMethods []string) grpc.UnaryServerInterceptor {
	public := make(map[string]struct{}, len(publicMethods))
	for _, m := range publicMethods {
		public[m] = struct{}{}
	}

	authInterceptor := auth.UnaryServerInterceptor(authFunc(tm))

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, isPublic := public[info.FullMethod]; isPublic {
			return handler(ctx, req)
		}
		return authInterceptor(ctx, req, info, handler)
	}
}

// authFunc extracts the bearer token from the request metadata, validates it
// via tm and injects the sub claim into the returned context.
func authFunc(tm TokenManager) auth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		token, err := auth.AuthFromMD(ctx, bearerScheme)
		if err != nil {
			// AuthFromMD already returns a codes.Unauthenticated status.
			return nil, err
		}

		sub, err := tm.Validate(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "não autenticado")
		}

		return context.WithValue(ctx, subKey, sub), nil
	}
}
