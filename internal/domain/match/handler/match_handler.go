// Package handler implements the gRPC presentation layer for the match domain.
// MatchHandler is a pure mapper: it translates between the generated proto
// messages and the service's domain types. It reads the authenticated user id
// from the context (anti-IDOR) and propagates service errors as codes.Internal
// without re-translating.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	matchv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/match/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/service"
)

// MatchService is the subset of the match domain service consumed by the
// handler. Declared here, in the consumer package, so the handler depends on a
// narrow interface rather than the concrete *service.MatchService.
type MatchService interface {
	ListUpcomingMatches(ctx context.Context, userID string) ([]service.Match, error)
}

// MatchHandler implements the generated MatchServiceServer by mapping proto
// requests to domain calls and domain results back to proto responses.
type MatchHandler struct {
	matchv1.UnimplementedMatchServiceServer
	svc MatchService
}

// NewMatchHandler constructs a MatchHandler backed by the given match service.
func NewMatchHandler(svc MatchService) *MatchHandler {
	return &MatchHandler{svc: svc}
}

// ListUpcomingMatches reads the authenticated user id from the context, calls
// the service and maps the result to a proto response. A missing subject yields
// codes.Unauthenticated. Service errors are mapped to codes.Internal (mapper
// puro — no re-translation of error codes).
func (h *MatchHandler) ListUpcomingMatches(
	ctx context.Context,
	_ *matchv1.ListUpcomingMatchesRequest,
) (*matchv1.ListUpcomingMatchesResponse, error) {
	sub, ok := interceptor.SubjectFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "não autenticado")
	}

	matches, err := h.svc.ListUpcomingMatches(ctx, sub)
	if err != nil {
		return nil, status.Error(codes.Internal, "erro interno")
	}

	protoMatches := make([]*matchv1.Match, 0, len(matches))
	for _, m := range matches {
		protoMatches = append(protoMatches, &matchv1.Match{
			Id:      m.ID,
			Kickoff: timestamppb.New(m.Kickoff),
			HomeTeam: &matchv1.MatchTeam{
				Id:      m.HomeTeam.ID,
				Name:    m.HomeTeam.Name,
				FlagUrl: m.HomeTeam.FlagURL,
				Code:    m.HomeTeam.Code,
			},
			AwayTeam: &matchv1.MatchTeam{
				Id:      m.AwayTeam.ID,
				Name:    m.AwayTeam.Name,
				FlagUrl: m.AwayTeam.FlagURL,
				Code:    m.AwayTeam.Code,
			},
			Stadium: m.Stadium,
			City:    m.City,
			Stage:   m.Stage,
		})
	}

	return &matchv1.ListUpcomingMatchesResponse{Matches: protoMatches}, nil
}
