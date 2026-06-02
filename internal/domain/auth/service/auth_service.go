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
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
)

// defaultBcryptCost is the bcrypt cost used in production when the constructor
// is called with cost 0. Cost 12 per Tech Spec §11.3.
const defaultBcryptCost = 12

// recoveryTTL is how long a temporary recovery password remains valid.
const recoveryTTL = 15 * time.Minute

// tempPasswordLength is the number of characters in a generated temporary
// password. tempPasswordAlphabet is the ambiguity-free set drawn from.
const tempPasswordLength = 16

const tempPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// Generic, field-agnostic messages. They must NOT reveal which credential
// failed, so e-mail existence cannot be inferred from the response (RN5).
const (
	msgInvalidNationalTeam   = "seleção favorita inválida"
	msgDuplicateNationalTeam = "seleção favorita duplicada"
	msgDuplicateEmail        = "e-mail já está em uso"
	msgInvalidCredentials    = "e-mail ou senha inválidos"
)

// msgRecoveryRequested is the generic, account-agnostic response always returned
// by RequestPasswordRecovery, so e-mail existence cannot be inferred (CA-02).
const msgRecoveryRequested = "Se o e-mail estiver cadastrado, enviaremos as instruções de recuperação."

const (
	recoveryEmailSubject = "Recuperação de acesso — Album Figurinhas da Copa do Mundo 2026"
	recoveryEmailBody    = "Recebemos um pedido de recuperação de acesso.\n\n" +
		"Sua senha temporária é: %s\n\n" +
		"Ela é válida por 15 minutos. Use-a para entrar e troque sua senha em seguida.\n\n" +
		"Se você não solicitou, ignore este e-mail."
)

// msgInvalidTempPassword is the single message returned when the temporary
// recovery password is missing, expired or incorrect. It must not distinguish
// which condition failed.
const msgInvalidTempPassword = "senha temporária inválida ou expirada"

const (
	passwordChangedEmailSubject = "Sua senha foi alterada — Album Figurinhas da Copa do Mundo 2026"
	passwordChangedEmailBody    = "Sua senha de acesso foi alterada.\n\n" +
		"Se foi você, nenhuma ação é necessária.\n\n" +
		"Se você não reconhece esta alteração, redefina seu acesso imediatamente."
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
	// TempPasswordHash is the bcrypt hash of the active recovery password, empty
	// when none is pending. TempPasswordExpiresAt is its expiration (zero when
	// none is pending).
	TempPasswordHash      string
	TempPasswordExpiresAt time.Time
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
	// SetTempPassword stores the bcrypt hash of a temporary recovery password
	// and its expiration for the user with the given id.
	SetTempPassword(ctx context.Context, id string, hash string, expiresAt time.Time) error
	// UpdatePassword replaces the permanent password hash for the user with the
	// given id (clearing any pending temporary password).
	UpdatePassword(ctx context.Context, id string, hash string) error
}

// EmailSender abstracts delivery of a plain-text e-mail. Declared in the
// consumer package (service); concrete senders live elsewhere and fx binds them
// to this interface. The signature is canonical — mirror it exactly.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
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

// LoginResult is the outcome of a successful login. PasswordChangeRequired is
// true when access was granted through an active temporary recovery password,
// signalling the client to force a password change.
type LoginResult struct {
	AccessToken            string
	ExpiresAt              time.Time
	PasswordChangeRequired bool
}

// compareFunc has the signature of bcrypt.CompareHashAndPassword. It is a
// constructor-injected dependency (defaulting to bcrypt) so that the
// timing-equalisation comparison is an observable boundary in tests, mirroring
// how clock.Clock is injected. This is not a test-only branch: production wires
// the real bcrypt function.
type compareFunc func(hashedPassword, password []byte) error

// genPasswordFunc produces a fresh temporary password. It is a
// constructor-injected boundary (defaulting to crypto/rand) so the generated
// value is controllable in tests, mirroring how compareHash and clock.Clock are
// injected. Production wires the real crypto/rand generator.
type genPasswordFunc func() (string, error)

// AuthService implements registration and login.
type AuthService struct {
	users           UserRepository
	nationalTeams   NationalTeamRepository
	tokens          TokenManager
	emailSender     EmailSender
	clk             clock.Clock
	logger          *zap.Logger
	bcryptCost      int
	compareHash     compareFunc
	genTempPassword genPasswordFunc

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
	emailSender EmailSender,
	clk clock.Clock,
	logger *zap.Logger,
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
		users:           users,
		nationalTeams:   nationalTeams,
		tokens:          tokens,
		emailSender:     emailSender,
		clk:             clk,
		logger:          logger,
		bcryptCost:      bcryptCost,
		compareHash:     bcrypt.CompareHashAndPassword,
		genTempPassword: generateTempPassword,
		dummyHash:       dummyHash,
	}, nil
}

// generateTempPassword is the production temporary-password generator: it draws
// tempPasswordLength characters uniformly from an ambiguity-free alphabet using
// crypto/rand.
func generateTempPassword() (string, error) {
	buf := make([]byte, tempPasswordLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, tempPasswordLength)
	for i, b := range buf {
		out[i] = tempPasswordAlphabet[int(b)%len(tempPasswordAlphabet)]
	}
	return string(out), nil
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

	// The permanent password is checked first: a match always grants normal
	// access (PasswordChangeRequired=false) even when a temporary password is
	// active or expired (RN10/CA-10).
	passwordChangeRequired := false
	if err := s.compareHash([]byte(user.PasswordHash), []byte(password)); err != nil {
		// The second comparison runs only when the permanent password failed AND
		// an active, non-expired temporary password exists — so login does not
		// pay a double bcrypt cost on every request (§20). The boundary uses
		// Before, so expires_at == now is treated as expired (CA-05).
		tempActive := user.TempPasswordHash != "" && s.clk.Now().Before(user.TempPasswordExpiresAt)
		if !tempActive || s.compareHash([]byte(user.TempPasswordHash), []byte(password)) != nil {
			return LoginResult{}, status.Error(codes.Unauthenticated, msgInvalidCredentials)
		}
		passwordChangeRequired = true
	}

	token, expiresAt, err := s.tokens.Generate(user.ID)
	if err != nil {
		return LoginResult{}, status.Error(codes.Internal, "erro interno")
	}

	return LoginResult{AccessToken: token, ExpiresAt: expiresAt, PasswordChangeRequired: passwordChangeRequired}, nil
}

// ChangePassword replaces the user's permanent password after validating the
// active temporary recovery password. userID comes from the token's sub
// (supplied by the handler), never from client input (anti-IDOR). The temporary
// password is checked strictly — it must exist, be unexpired (Before boundary,
// matching Login) and match — so the flow only works post-recovery and never
// accepts the current password. UpdatePassword clears the temp columns in the
// same operation, and the notification e-mail is best-effort: it is sent after
// the change is persisted, and a delivery failure does not undo the change.
func (s *AuthService) ChangePassword(ctx context.Context, userID, tempPassword, newPassword string) error {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return status.Error(codes.Internal, "erro interno")
	}

	tempValid := user.TempPasswordHash != "" &&
		s.clk.Now().Before(user.TempPasswordExpiresAt) &&
		s.compareHash([]byte(user.TempPasswordHash), []byte(tempPassword)) == nil
	if !tempValid {
		return status.Error(codes.InvalidArgument, msgInvalidTempPassword)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return status.Error(codes.Internal, "erro interno")
	}

	if err := s.users.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return status.Error(codes.Internal, "erro interno")
	}

	if err := s.emailSender.Send(ctx, user.Email, passwordChangedEmailSubject, passwordChangedEmailBody); err != nil {
		s.logger.Warn("password change: notification e-mail delivery failed",
			zap.String("to", user.Email),
			zap.String("subject", passwordChangedEmailSubject),
			zap.Error(err),
		)
	}

	return nil
}

// RequestPasswordRecovery starts the password-recovery flow for the given
// e-mail. It returns immediately with a generic, account-agnostic message
// (always OK) so e-mail existence cannot be inferred from the response (CA-02),
// and performs the actual work asynchronously. The detached context survives
// the RPC returning (§10.2).
func (s *AuthService) RequestPasswordRecovery(ctx context.Context, email string) (string, error) {
	go s.processRecovery(context.WithoutCancel(ctx), email)
	return msgRecoveryRequested, nil
}

// processRecovery is the synchronous core of the recovery flow (the testable
// seam behind RequestPasswordRecovery's goroutine). It is best-effort: every
// branch returns nil because there is no client response left to change. An
// unknown e-mail is silent (CA-02); an e-mail-delivery failure is logged
// without the temporary password (RN4/CA-03) and does not undo the stored hash.
func (s *AuthService) processRecovery(ctx context.Context, email string) error {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		s.logger.Error("password recovery: lookup failed", zap.Error(err))
		return nil
	}

	tempPassword, err := s.genTempPassword()
	if err != nil {
		s.logger.Error("password recovery: temp password generation failed", zap.Error(err))
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), s.bcryptCost)
	if err != nil {
		s.logger.Error("password recovery: hashing failed", zap.Error(err))
		return nil
	}

	if err := s.users.SetTempPassword(ctx, user.ID, string(hash), s.clk.Now().Add(recoveryTTL)); err != nil {
		s.logger.Error("password recovery: persisting temp password failed", zap.Error(err))
		return nil
	}

	body := fmt.Sprintf(recoveryEmailBody, tempPassword)
	if err := s.emailSender.Send(ctx, email, recoveryEmailSubject, body); err != nil {
		s.logger.Warn("password recovery: e-mail delivery failed",
			zap.String("to", email),
			zap.String("subject", recoveryEmailSubject),
			zap.Error(err),
		)
		return nil
	}

	return nil
}
