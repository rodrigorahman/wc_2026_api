// Package handler implements the gRPC presentation layer for the
// nationalteam domain. NationalTeamHandler is a pure mapper: it translates
// between the generated proto messages and the service's domain types and
// propagates the service's errors verbatim. The error-to-gRPC-status mapping
// is owned by the service, so the handler never re-translates error codes.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
	nationalteamv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/nationalteam/v1"
)

// NationalTeamService is the subset of the nationalteam domain service
// consumed by the handler. Declared here, in the consumer package, so the
// handler depends on a narrow interface rather than the concrete
// *service.NationalTeamService.
type NationalTeamService interface {
	ListNationalTeams(ctx context.Context) ([]service.NationalTeam, error)
}

// NationalTeamHandler implements the generated NationalTeamServiceServer by
// mapping proto requests to domain calls and domain results back to proto
// responses.
type NationalTeamHandler struct {
	nationalteamv1.UnimplementedNationalTeamServiceServer
	svc NationalTeamService
}

// NewNationalTeamHandler constructs a NationalTeamHandler backed by the
// given nationalteam service.
func NewNationalTeamHandler(svc NationalTeamService) *NationalTeamHandler {
	return &NationalTeamHandler{svc: svc}
}

// ListNationalTeams maps the domain slice to a proto ListNationalTeamsResponse.
// This RPC is public (no auth required). Service errors are propagated
// unchanged.
func (h *NationalTeamHandler) ListNationalTeams(
	ctx context.Context,
	_ *nationalteamv1.ListNationalTeamsRequest,
) (*nationalteamv1.ListNationalTeamsResponse, error) {
	teams, err := h.svc.ListNationalTeams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "erro interno")
	}

	protoTeams := make([]*nationalteamv1.NationalTeam, 0, len(teams))
	for _, t := range teams {
		protoTeams = append(protoTeams, &nationalteamv1.NationalTeam{
			Id:      t.ID,
			Name:    t.Name,
			FlagUrl: t.FlagURL,
		})
	}

	return &nationalteamv1.ListNationalTeamsResponse{NationalTeams: protoTeams}, nil
}
