package interceptor_test

import (
	"context"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
)

// spyHandler records whether the wrapped gRPC handler was invoked.
type spyHandler struct {
	called bool
}

func (h *spyHandler) handle(ctx context.Context, _ any) (any, error) {
	h.called = true
	return &authv1.RegisterResponse{}, nil
}

// validRegisterRequest returns a request that satisfies every buf.validate rule
// so individual tests can mutate exactly one field.
func validRegisterRequest() *authv1.RegisterRequest {
	return &authv1.RegisterRequest{
		FullName:        "Alice Souza",
		Email:           "alice@example.com",
		Password:        "supersecret",
		NationalTeamIds: []string{"11111111-2222-3333-4444-555555555555"},
	}
}

// CT-003 / CT-005: protovalidate rejects invalid requests with InvalidArgument
// before the handler runs.
func TestProtoValidate_RejectsInvalidRequest(t *testing.T) {
	validator, err := protovalidate.New()
	require.NoError(t, err)

	tests := []struct {
		name string
		req  *authv1.RegisterRequest
	}{
		{
			name: "missing_required_full_name", // CT-003
			req: func() *authv1.RegisterRequest {
				r := validRegisterRequest()
				r.FullName = ""
				return r
			}(),
		},
		{
			name: "invalid_email_format", // CT-005
			req: func() *authv1.RegisterRequest {
				r := validRegisterRequest()
				r.Email = "abc"
				return r
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spy := &spyHandler{}
			intercept := interceptor.NewProtoValidateUnaryInterceptor(validator)

			_, err := intercept(
				context.Background(),
				tc.req,
				&grpc.UnaryServerInfo{FullMethod: authv1.AuthService_Register_FullMethodName},
				spy.handle,
			)

			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.False(t, spy.called, "handler must not run when validation fails")
		})
	}
}

// A request satisfying every rule reaches the handler unchanged.
func TestProtoValidate_ValidRequest_CallsHandler(t *testing.T) {
	validator, err := protovalidate.New()
	require.NoError(t, err)

	spy := &spyHandler{}
	intercept := interceptor.NewProtoValidateUnaryInterceptor(validator)

	resp, err := intercept(
		context.Background(),
		validRegisterRequest(),
		&grpc.UnaryServerInfo{FullMethod: authv1.AuthService_Register_FullMethodName},
		spy.handle,
	)

	require.NoError(t, err)
	require.True(t, spy.called, "handler must run for a valid request")
	require.IsType(t, &authv1.RegisterResponse{}, resp)
}
