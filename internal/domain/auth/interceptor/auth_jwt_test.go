package interceptor_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
)

const (
	jwtTestSecret = "test-secret-at-least-32-bytes-long-xxxx"
	jwtTestTTL    = time.Hour
	jwtTestUserID = "11111111-2222-3333-4444-555555555555"
)

// fixedClock is a deterministic, advanceable Clock for token timing control.
type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time { return c.now }

func (c *fixedClock) advance(d time.Duration) { c.now = c.now.Add(d) }

// authCapture records whether the protected handler ran and the sub it observed
// in the context the interceptor produced.
type authCapture struct {
	called bool
	sub    string
	subOK  bool
}

func (h *authCapture) handle(ctx context.Context, _ any) (any, error) {
	h.called = true
	h.sub, h.subOK = interceptor.SubjectFromContext(ctx)
	return &authv1.GetMeResponse{}, nil
}

// protectedInfo is the UnaryServerInfo for the protected GetMe RPC.
func protectedInfo() *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: authv1.AuthService_GetMe_FullMethodName}
}

// publicMethods lists the RPCs that bypass authentication.
func publicMethods() []string {
	return []string{
		authv1.AuthService_Register_FullMethodName,
		authv1.AuthService_Login_FullMethodName,
	}
}

// bearerCtx builds an incoming-metadata context carrying a Bearer token.
func bearerCtx(token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

// CT-010: a valid, unexpired token runs the handler with sub in the context.
func TestJWT_ValidToken_AllowsAccess(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	clk := &fixedClock{now: now}
	tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, clk)

	tokenStr, _, err := tm.Generate(jwtTestUserID)
	require.NoError(t, err)

	cap := &authCapture{}
	intercept := interceptor.NewAuthJWTUnaryInterceptor(tm, publicMethods())

	_, err = intercept(bearerCtx(tokenStr), &authv1.GetMeRequest{}, protectedInfo(), cap.handle)

	require.NoError(t, err)
	require.True(t, cap.called, "handler must run for a valid token")
	require.True(t, cap.subOK, "sub must be present in the context")
	require.Equal(t, jwtTestUserID, cap.sub)
}

// CT-011/012/013/014: invalid tokens deny access with Unauthenticated and never
// reach the handler.
func TestJWT_InvalidTokens_Rejected(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	exp := now.Add(jwtTestTTL)

	tests := []struct {
		name string
		// ctx builds the incoming context; tm is the validator wired into the
		// interceptor with a clock the test controls.
		buildCtx func(t *testing.T) (context.Context, interceptor.TokenManager)
	}{
		{
			name: "expired_token", // CT-011-interceptor
			buildCtx: func(t *testing.T) (context.Context, interceptor.TokenManager) {
				clk := &fixedClock{now: now}
				tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, clk)
				tokenStr, _, err := tm.Generate(jwtTestUserID)
				require.NoError(t, err)
				clk.advance(jwtTestTTL + time.Minute)
				return bearerCtx(tokenStr), tm
			},
		},
		{
			name: "invalid_signature", // CT-012
			buildCtx: func(t *testing.T) (context.Context, interceptor.TokenManager) {
				clk := &fixedClock{now: now}
				issuer := token.NewTokenManager([]byte("another-secret-at-least-32-bytes-long"), jwtTestTTL, clk)
				tokenStr, _, err := issuer.Generate(jwtTestUserID)
				require.NoError(t, err)
				validator := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, &fixedClock{now: now})
				return bearerCtx(tokenStr), validator
			},
		},
		{
			name: "missing_token", // CT-013
			buildCtx: func(t *testing.T) (context.Context, interceptor.TokenManager) {
				tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, &fixedClock{now: now})
				return context.Background(), tm
			},
		},
		{
			name: "alg_none", // CT-014
			buildCtx: func(t *testing.T) (context.Context, interceptor.TokenManager) {
				tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, &fixedClock{now: now})
				return bearerCtx(craftNoneToken(t, jwtTestUserID, exp)), tm
			},
		},
		{
			name: "alg_RS256", // CT-014
			buildCtx: func(t *testing.T) (context.Context, interceptor.TokenManager) {
				tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, &fixedClock{now: now})
				return bearerCtx(craftRS256Token(t, jwtTestUserID, exp)), tm
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, tm := tc.buildCtx(t)

			cap := &authCapture{}
			intercept := interceptor.NewAuthJWTUnaryInterceptor(tm, publicMethods())

			_, err := intercept(ctx, &authv1.GetMeRequest{}, protectedInfo(), cap.handle)

			require.Error(t, err)
			require.Equal(t, codes.Unauthenticated, status.Code(err))
			require.False(t, cap.called, "handler must not run when authentication fails")
		})
	}
}

// A public RPC bypasses authentication even with no token present.
func TestJWT_PublicMethod_BypassesAuth(t *testing.T) {
	tm := token.NewTokenManager([]byte(jwtTestSecret), jwtTestTTL, &fixedClock{now: time.Now()})

	cap := &authCapture{}
	intercept := interceptor.NewAuthJWTUnaryInterceptor(tm, publicMethods())

	_, err := intercept(
		context.Background(),
		&authv1.RegisterRequest{},
		&grpc.UnaryServerInfo{FullMethod: authv1.AuthService_Register_FullMethodName},
		cap.handle,
	)

	require.NoError(t, err)
	require.True(t, cap.called, "public RPC must run without a token")
	require.False(t, cap.subOK, "no sub is injected for a public RPC")
}

// --- helpers: manually crafted tokens (NOT via TokenManager.Generate) ---

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// craftNoneToken builds an unsigned "alg":"none" JWT by hand.
func craftNoneToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	require.NoError(t, err)
	claims, err := json.Marshal(map[string]any{"sub": sub, "exp": exp.Unix()})
	require.NoError(t, err)
	return strings.Join([]string{b64(header), b64(claims), ""}, ".")
}

// craftRS256Token builds a genuinely RS256-signed JWT using an ephemeral RSA key.
func craftRS256Token(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": sub,
		"exp": exp.Unix(),
	})
	signed, err := tok.SignedString(key)
	require.NoError(t, err)
	return signed
}
