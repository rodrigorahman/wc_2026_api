// Package service implements the authentication domain logic: user
// registration (with bcrypt password hashing and national-team validation)
// and login (with constant-message, timing-equalised credential checking to
// defend against e-mail enumeration).
//
// Following the project's idiomatic interface placement, the repository and
// token-manager interfaces this service depends on are declared HERE, in the
// consumer package. The repository packages provide concrete types that
// satisfy them, and fx binds concrete to interface in the wiring.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
)

// defaultBcryptCost is the bcrypt cost used in production when the constructor
// is called with cost 0. Cost 12 per Tech Spec §11.3.
const defaultBcryptCost = 12

// Generic, field-agnostic messages. They must NOT reveal which credential
// failed, so e-mail existence cannot be inferred from the response (RN5).
const (
	msgInvalidNationalTeam   = "seleção favorita inválida"
	msgDuplicateNationalTeam = "seleção favorita duplicada"
	msgDuplicateEmail        = "e-mail já está em uso"
	msgInvalidCredentials    = "e-mail ou senha inválidos"
)

// Domain sentinel errors. Repositories signal "not found" with these so the
// service can branch via errors.Is without coupling to a storage-specific
// error type.
var (
	// ErrUserNotFound is returned by UserRepository.GetUserByEmail when no user
	// matches the given e-mail.
	ErrUserNotFound = errors.New("user not found")
	// ErrNationalTeamNotFound is returned by NationalTeamRepository.GetNationalTeamByID
	// when no national team matches the given id.
	ErrNationalTeamNotFound = errors.New("national team not found")
)

// User is the domain representation of a registered user as consumed by this
// service. Repositories map their storage rows into this shape.
type User struct {
	ID           string
	FullName     string
	Email        string
	PasswordHash string
	// NationalTeamIDs are the ids of the user's favourite national teams (1..3).
	NationalTeamIDs []string
}

// UserRepository abstracts persistence of users. Declared in the consumer
// package (service) per the project's interface-placement convention.
type UserRepository interface {
	// CreateUser persists a new user. The PasswordHash field must already
	// contain a bcrypt hash; the plain password never reaches this layer.
	CreateUser(ctx context.Context, user User) error
	// GetUserByEmail returns the user with the given e-mail, or ErrUserNotFound
	// if none exists.
	GetUserByEmail(ctx context.Context, email string) (User, error)
	// GetUserByID returns the user with the given id, or ErrUserNotFound if
	// none exists.
	GetUserByID(ctx context.Context, id string) (User, error)
}

// NationalTeamRepository abstracts read access to national teams.
type NationalTeamRepository interface {
	// GetNationalTeamByID returns ErrNationalTeamNotFound when the id does not
	// exist. The returned name is unused by this service today; only existence
	// matters for registration validation.
	GetNationalTeamByID(ctx context.Context, id string) (string, error)
}

// TokenManager abstracts issuing access tokens for authenticated users.
type TokenManager interface {
	// Generate issues a signed token for userID and returns it together with
	// its expiration time.
	Generate(userID string) (string, time.Time, error)
}

// RegisterParams carries the data required to register a new user.
type RegisterParams struct {
	FullName        string
	Email           string
	Password        string
	NationalTeamIDs []string
}

// LoginResult is the outcome of a successful login.
type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
}

// compareFunc has the signature of bcrypt.CompareHashAndPassword. It is a
// constructor-injected dependency (defaulting to bcrypt) so that the
// timing-equalisation comparison is an observable boundary in tests, mirroring
// how clock.Clock is injected. This is not a test-only branch: production wires
// the real bcrypt function.
type compareFunc func(hashedPassword, password []byte) error

// AuthService implements registration and login.
type AuthService struct {
	users         UserRepository
	nationalTeams NationalTeamRepository
	tokens        TokenManager
	clk           clock.Clock
	bcryptCost    int
	compareHash   compareFunc

	// dummyHash is a fixed bcrypt hash compared against in the "e-mail not
	// found" login path to equalise response time with the "wrong password"
	// path, closing the timing side-channel for e-mail enumeration (§11.3).
	dummyHash []byte
}

// NewAuthService constructs an AuthService. bcryptCost defaults to 12 when 0 is
// passed (allowing CI to lower it if hashing exceeds the time budget). The
// dummy hash used for login timing-equalisation is generated once here.
func NewAuthService(
	users UserRepository,
	nationalTeams NationalTeamRepository,
	tokens TokenManager,
	clk clock.Clock,
	bcryptCost int,
) (*AuthService, error) {
	if bcryptCost == 0 {
		bcryptCost = defaultBcryptCost
	}

	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing-equalisation"), bcryptCost)
	if err != nil {
		return nil, err
	}

	return &AuthService{
		users:         users,
		nationalTeams: nationalTeams,
		tokens:        tokens,
		clk:           clk,
		bcryptCost:    bcryptCost,
		compareHash:   bcrypt.CompareHashAndPassword,
		dummyHash:     dummyHash,
	}, nil
}

// Register validates the chosen national team and e-mail uniqueness, hashes the
// password with bcrypt, assigns a UUID v4, persists the user and returns its id.
func (s *AuthService) Register(ctx context.Context, params RegisterParams) (string, error) {
	seen := make(map[string]struct{}, len(params.NationalTeamIDs))
	for _, teamID := range params.NationalTeamIDs {
		if _, dup := seen[teamID]; dup {
			return "", status.Error(codes.InvalidArgument, msgDuplicateNationalTeam)
		}
		seen[teamID] = struct{}{}

		if _, err := s.nationalTeams.GetNationalTeamByID(ctx, teamID); err != nil {
			if errors.Is(err, ErrNationalTeamNotFound) {
				return "", status.Error(codes.InvalidArgument, msgInvalidNationalTeam)
			}
			return "", status.Error(codes.Internal, "erro interno")
		}
	}

	switch _, err := s.users.GetUserByEmail(ctx, params.Email); {
	case err == nil:
		return "", status.Error(codes.AlreadyExists, msgDuplicateEmail)
	case errors.Is(err, ErrUserNotFound):
		// E-mail is free; continue.
	default:
		return "", status.Error(codes.Internal, "erro interno")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), s.bcryptCost)
	if err != nil {
		return "", status.Error(codes.Internal, "erro interno")
	}

	userID := uuid.NewString()

	user := User{
		ID:              userID,
		FullName:        params.FullName,
		Email:           params.Email,
		PasswordHash:    string(hash),
		NationalTeamIDs: params.NationalTeamIDs,
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return "", status.Error(codes.Internal, "erro interno")
	}

	return userID, nil
}

// GetUser returns the registered user with the given id. It maps an unknown id
// to codes.NotFound and any other repository failure to codes.Internal, so the
// handler can propagate the error verbatim (the service owns the status mapping).
func (s *AuthService) GetUser(ctx context.Context, id string) (User, error) {
	user, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, status.Error(codes.NotFound, "usuário não encontrado")
		}
		return User{}, status.Error(codes.Internal, "erro interno")
	}

	return user, nil
}

// Login authenticates a user by e-mail and password. A failure for either an
// unknown e-mail or a wrong password returns the same generic Unauthenticated
// message. In the "unknown e-mail" path a bcrypt comparison against a dummy
// hash is still performed to equalise response time (anti-enumeration, RN5).
func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Equalise timing: run the same bcrypt comparison cost as the
			// "wrong password" path before returning the generic error.
			_ = s.compareHash(s.dummyHash, []byte(password))
			return LoginResult{}, status.Error(codes.Unauthenticated, msgInvalidCredentials)
		}
		return LoginResult{}, status.Error(codes.Internal, "erro interno")
	}

	if err := s.compareHash([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, status.Error(codes.Unauthenticated, msgInvalidCredentials)
	}

	token, expiresAt, err := s.tokens.Generate(user.ID)
	if err != nil {
		return LoginResult{}, status.Error(codes.Internal, "erro interno")
	}

	return LoginResult{AccessToken: token, ExpiresAt: expiresAt}, nil
}
