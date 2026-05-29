// Package handler implements the gRPC presentation layer for the auth domain.
// AuthHandler is a pure mapper: it translates between the generated proto
// messages and the service's domain types and propagates the service's errors
// verbatim. The error-to-gRPC-status mapping is owned by the service (§10.1),
// so the handler never re-translates error codes.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
)

// AuthService is the subset of the auth domain service consumed by the handler.
// Declared here, in the consumer package, so the handler depends on a narrow
// interface rather than the concrete *service.AuthService.
type AuthService interface {
	Register(ctx context.Context, params service.RegisterParams) (string, error)
	Login(ctx context.Context, email, password string) (service.LoginResult, error)
	GetUser(ctx context.Context, id string) (service.User, error)
}

// AuthHandler implements the generated AuthServiceServer by mapping proto
// requests to domain calls and domain results back to proto responses.
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc AuthService
}

// NewAuthHandler constructs an AuthHandler backed by the given auth service.
func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register maps RegisterRequest to a domain RegisterParams, delegates to the
// service and returns the new user id. Service errors (already gRPC statuses)
// are propagated unchanged.
func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	userID, err := h.svc.Register(ctx, service.RegisterParams{
		FullName:        req.GetFullName(),
		Email:           req.GetEmail(),
		Password:        req.GetPassword(),
		NationalTeamIDs: req.GetNationalTeamIds(),
	})
	if err != nil {
		return nil, err
	}

	return &authv1.RegisterResponse{UserId: userID}, nil
}

// Login maps LoginRequest to the service Login call and converts the resulting
// expiration time to a protobuf Timestamp. Service errors are propagated
// unchanged.
func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	result, err := h.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken: result.AccessToken,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}, nil
}

// GetMe reads the authenticated user id (sub claim) injected into the context
// by the JWT interceptor (T10), loads the user and returns its data. A missing
// subject means the RPC reached the handler without authentication, which is an
// internal wiring fault (the interceptor must reject unauthenticated calls
// before the handler runs). Service errors are propagated unchanged.
func (h *AuthHandler) GetMe(ctx context.Context, _ *authv1.GetMeRequest) (*authv1.GetMeResponse, error) {
	sub, ok := interceptor.SubjectFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "não autenticado")
	}

	user, err := h.svc.GetUser(ctx, sub)
	if err != nil {
		return nil, err
	}

	return &authv1.GetMeResponse{
		UserId:          user.ID,
		FullName:        user.FullName,
		Email:           user.Email,
		NationalTeamIds: user.NationalTeamIDs,
	}, nil
}
