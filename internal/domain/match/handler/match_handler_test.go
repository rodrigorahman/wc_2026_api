package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	matchv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/match/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/service"
)

// --- Manual mock -------------------------------------------------------------

type matchServiceMock struct {
	listCalled bool
	listUserID string
	listFn     func(ctx context.Context, userID string) ([]service.Match, error)
}

func (m *matchServiceMock) ListUpcomingMatches(ctx context.Context, userID string) ([]service.Match, error) {
	m.listCalled = true
	m.listUserID = userID
	return m.listFn(ctx, userID)
}

// --- Helpers -----------------------------------------------------------------

const (
	authTestSecret = "test-secret-at-least-32-bytes-long-xxxx"
	authTestTTL    = time.Hour
)

// fixedClock is a deterministic Clock so the issued token's exp is stable.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// ctxWithSub returns a context whose authenticated subject equals sub, derived
// the same way production does: a real HS256 token is issued for sub and then
// run through the real JWT interceptor, which validates it and injects the sub
// under the package-private subKey. This keeps subKey encapsulated and avoids
// any test-only injection symbol in production code.
func ctxWithSub(t *testing.T, sub string) context.Context {
	t.Helper()

	clk := fixedClock{now: time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)}
	tm := token.NewTokenManager([]byte(authTestSecret), authTestTTL, clk)

	signed, _, err := tm.Generate(sub)
	require.NoError(t, err)

	intercept := interceptor.NewAuthJWTUnaryInterceptor(tm, nil)
	incoming := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("authorization", "Bearer "+signed),
	)

	var authCtx context.Context
	_, err = intercept(
		incoming,
		&matchv1.ListUpcomingMatchesRequest{},
		&grpc.UnaryServerInfo{FullMethod: matchv1.MatchService_ListUpcomingMatches_FullMethodName},
		func(ctx context.Context, _ any) (any, error) {
			authCtx = ctx
			return &authv1.GetMeResponse{}, nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, authCtx, "interceptor must invoke the handler for a valid token")

	return authCtx
}

// --- Tests -------------------------------------------------------------------

// T5-CT-001: context without subject must return Unauthenticated and must NOT
// call the service (anti-IDOR guard).
func TestListUpcoming_NoSubject_Unauthenticated(t *testing.T) {
	mock := &matchServiceMock{}
	h := handler.NewMatchHandler(mock)

	resp, err := h.ListUpcomingMatches(context.Background(), &matchv1.ListUpcomingMatchesRequest{})

	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, mock.listCalled, "service must NOT be called when subject is absent (anti-IDOR)")
}

// T5-CT-002: context with valid subject must map domain matches to proto,
// including kickoff timestamp and all team fields.
func TestListUpcoming_MapsDomainToProto(t *testing.T) {
	kickoff1 := time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC)
	kickoff2 := time.Date(2026, 6, 15, 21, 0, 0, 0, time.UTC)

	domainMatches := []service.Match{
		{
			ID:      "match-001",
			Kickoff: kickoff1,
			HomeTeam: service.MatchTeam{
				ID:      "team-bra",
				Name:    "Brasil",
				FlagURL: "https://flagcdn.com/w320/br.png",
				Code:    "BRA",
			},
			AwayTeam: service.MatchTeam{
				ID:      "team-arg",
				Name:    "Argentina",
				FlagURL: "https://flagcdn.com/w320/ar.png",
				Code:    "ARG",
			},
			Stadium: "MetLife Stadium",
			City:    "East Rutherford",
			Stage:   "Group Stage",
		},
		{
			ID:      "match-002",
			Kickoff: kickoff2,
			HomeTeam: service.MatchTeam{
				ID:      "team-fra",
				Name:    "França",
				FlagURL: "https://flagcdn.com/w320/fr.png",
				Code:    "FRA",
			},
			AwayTeam: service.MatchTeam{
				ID:      "team-bra",
				Name:    "Brasil",
				FlagURL: "https://flagcdn.com/w320/br.png",
				Code:    "BRA",
			},
			Stadium: "AT&T Stadium",
			City:    "Arlington",
			Stage:   "Knockout",
		},
	}

	mock := &matchServiceMock{
		listFn: func(_ context.Context, _ string) ([]service.Match, error) {
			return domainMatches, nil
		},
	}
	h := handler.NewMatchHandler(mock)

	resp, err := h.ListUpcomingMatches(ctxWithSub(t, "user-abc"), &matchv1.ListUpcomingMatchesRequest{})

	require.NoError(t, err)
	require.True(t, mock.listCalled, "handler must call ListUpcomingMatches on the service")
	require.NotNil(t, resp)
	require.Len(t, resp.GetMatches(), 2)

	got0 := resp.GetMatches()[0]
	require.Equal(t, "match-001", got0.GetId())
	require.True(t, kickoff1.Equal(got0.GetKickoff().AsTime()), "kickoff must round-trip via timestamppb")
	require.Equal(t, "team-bra", got0.GetHomeTeam().GetId())
	require.Equal(t, "Brasil", got0.GetHomeTeam().GetName())
	require.Equal(t, "https://flagcdn.com/w320/br.png", got0.GetHomeTeam().GetFlagUrl())
	require.Equal(t, "BRA", got0.GetHomeTeam().GetCode())
	require.Equal(t, "team-arg", got0.GetAwayTeam().GetId())
	require.Equal(t, "ARG", got0.GetAwayTeam().GetCode())
	require.Equal(t, "MetLife Stadium", got0.GetStadium())
	require.Equal(t, "East Rutherford", got0.GetCity())
	require.Equal(t, "Group Stage", got0.GetStage())

	got1 := resp.GetMatches()[1]
	require.Equal(t, "match-002", got1.GetId())
	require.True(t, kickoff2.Equal(got1.GetKickoff().AsTime()), "kickoff must round-trip via timestamppb")
	require.Equal(t, "FRA", got1.GetHomeTeam().GetCode())
	require.Equal(t, "BRA", got1.GetAwayTeam().GetCode())
}

// T5-CT-005: the subject from context is forwarded as userID to the service
// (anti-IDOR: userID comes from token, never from the empty request).
func TestListUpcoming_PassesSubjectAsUserID(t *testing.T) {
	mock := &matchServiceMock{
		listFn: func(_ context.Context, _ string) ([]service.Match, error) {
			return []service.Match{}, nil
		},
	}
	h := handler.NewMatchHandler(mock)

	_, err := h.ListUpcomingMatches(ctxWithSub(t, "user-known-id"), &matchv1.ListUpcomingMatchesRequest{})

	require.NoError(t, err)
	require.Equal(t, "user-known-id", mock.listUserID, "userID forwarded to service must equal the sub from context")
}

// T5-CT-003: service error must yield codes.Internal and nil response (mapper
// puro — no error code re-translation).
func TestListUpcoming_ServiceError_Internal(t *testing.T) {
	mock := &matchServiceMock{
		listFn: func(_ context.Context, _ string) ([]service.Match, error) {
			return nil, errors.New("database unavailable")
		},
	}
	h := handler.NewMatchHandler(mock)

	resp, err := h.ListUpcomingMatches(ctxWithSub(t, "user-abc"), &matchv1.ListUpcomingMatchesRequest{})

	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

// T5-CT-004: empty list from the service must return NoError with an empty
// (non-nil) matches slice.
func TestListUpcoming_EmptyList_NoError(t *testing.T) {
	mock := &matchServiceMock{
		listFn: func(_ context.Context, _ string) ([]service.Match, error) {
			return []service.Match{}, nil
		},
	}
	h := handler.NewMatchHandler(mock)

	resp, err := h.ListUpcomingMatches(ctxWithSub(t, "user-abc"), &matchv1.ListUpcomingMatchesRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.GetMatches(), 0)
}
