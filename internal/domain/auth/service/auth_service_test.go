package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
)

// --- Test constants -------------------------------------------------------

const (
	testCost            = bcrypt.MinCost // keep hashing fast in CI; behaviour is cost-independent.
	testEmail           = "user@example.com"
	testPassword        = "segredo123"
	testFullName        = "Maria Silva"
	testNationalTeamID  = "11111111-1111-1111-1111-111111111111"
	testNationalTeamID2 = "33333333-3333-3333-3333-333333333333"
	testNationalTeamID3 = "44444444-4444-4444-4444-444444444444"
	testUserID          = "22222222-2222-2222-2222-222222222222"
	testToken           = "signed.jwt.token"
)

// --- Manual mocks ---------------------------------------------------------

type userRepoMock struct {
	getByEmailFn      func(ctx context.Context, email string) (service.User, error)
	getByIDFn         func(ctx context.Context, id string) (service.User, error)
	createFn          func(ctx context.Context, u service.User) error
	setTempPasswordFn func(ctx context.Context, id, hash string, expiresAt time.Time) error
	updatePasswordFn  func(ctx context.Context, id, hash string) error

	createdUser   service.User
	createCalled  bool
	getEmailCalls []string
	getIDCalls    []string

	setTempCalled    bool
	setTempID        string
	setTempHash      string
	setTempExpiresAt time.Time

	updateCalled bool
	updateID     string
	updateHash   string
}

func (m *userRepoMock) GetUserByEmail(ctx context.Context, email string) (service.User, error) {
	m.getEmailCalls = append(m.getEmailCalls, email)
	return m.getByEmailFn(ctx, email)
}

func (m *userRepoMock) GetUserByID(ctx context.Context, id string) (service.User, error) {
	m.getIDCalls = append(m.getIDCalls, id)
	return m.getByIDFn(ctx, id)
}

func (m *userRepoMock) CreateUser(ctx context.Context, u service.User) error {
	m.createCalled = true
	m.createdUser = u
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}

func (m *userRepoMock) SetTempPassword(ctx context.Context, id, hash string, expiresAt time.Time) error {
	m.setTempCalled = true
	m.setTempID = id
	m.setTempHash = hash
	m.setTempExpiresAt = expiresAt
	if m.setTempPasswordFn != nil {
		return m.setTempPasswordFn(ctx, id, hash, expiresAt)
	}
	return nil
}

func (m *userRepoMock) UpdatePassword(ctx context.Context, id, hash string) error {
	m.updateCalled = true
	m.updateID = id
	m.updateHash = hash
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, hash)
	}
	return nil
}

type emailSenderMock struct {
	err error

	sendCalled bool
	toArg      string
	subjectArg string
	bodyArg    string
}

func (m *emailSenderMock) Send(_ context.Context, to, subject, body string) error {
	m.sendCalled = true
	m.toArg = to
	m.subjectArg = subject
	m.bodyArg = body
	return m.err
}

type nationalTeamRepoMock struct {
	getByIDFn func(ctx context.Context, id string) (string, error)

	getIDCalls []string
}

func (m *nationalTeamRepoMock) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	m.getIDCalls = append(m.getIDCalls, id)
	return m.getByIDFn(ctx, id)
}

type tokenManagerMock struct {
	generateFn func(userID string) (string, time.Time, error)

	generatedFor string
}

func (m *tokenManagerMock) Generate(userID string) (string, time.Time, error) {
	m.generatedFor = userID
	return m.generateFn(userID)
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// --- Helpers --------------------------------------------------------------

// teamFound makes GetNationalTeamByID succeed.
func teamFound() *nationalTeamRepoMock {
	return &nationalTeamRepoMock{
		getByIDFn: func(context.Context, string) (string, error) { return "Brasil", nil },
	}
}

// hashOf produces a bcrypt hash of password at the test cost.
func hashOf(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), testCost)
	require.NoError(t, err)
	return string(h)
}

func newService(
	t *testing.T,
	users service.UserRepository,
	teams service.NationalTeamRepository,
	tokens service.TokenManager,
) *service.AuthService {
	t.Helper()
	svc, err := service.NewAuthService(users, teams, tokens, &emailSenderMock{}, fixedClock{now: time.Now()}, zap.NewNop(), testCost)
	require.NoError(t, err)
	return svc
}

// --- Register -------------------------------------------------------------

// CT-036: a valid registration with one selection persists a user and returns a
// non-empty id.
// INVARIANT: Register with a valid team and free e-mail persists exactly one
// user (whose stored id is the returned id) carrying the chosen selection and
// never stores the plain password.
func TestRegister_Success(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	gotID, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID},
	})

	require.NoError(t, err)
	require.NotEmpty(t, gotID)
	require.True(t, users.createCalled, "CreateUser must be called")
	// The returned id is the one actually persisted (not a value the test set).
	require.Equal(t, users.createdUser.ID, gotID)
	require.Equal(t, testEmail, users.createdUser.Email)
	require.Equal(t, testFullName, users.createdUser.FullName)
	require.Equal(t, []string{testNationalTeamID}, users.createdUser.NationalTeamIDs)
}

// CT-037: a valid registration with three distinct selections persists all of
// them and returns a non-empty id.
func TestRegister_ThreeSelections_Success(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	teamIDs := []string{testNationalTeamID, testNationalTeamID2, testNationalTeamID3}
	gotID, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: teamIDs,
	})

	require.NoError(t, err)
	require.NotEmpty(t, gotID)
	require.True(t, users.createCalled)
	require.Equal(t, users.createdUser.ID, gotID)
	require.Equal(t, teamIDs, users.createdUser.NationalTeamIDs)
}

// CT-038: registrations of 1, 2 and 3 valid selections each persist exactly the
// chosen selections.
func TestRegister_SelectionCounts_Persisted(t *testing.T) {
	cases := map[string][]string{
		"one":   {testNationalTeamID},
		"two":   {testNationalTeamID, testNationalTeamID2},
		"three": {testNationalTeamID, testNationalTeamID2, testNationalTeamID3},
	}
	for name, teamIDs := range cases {
		t.Run(name, func(t *testing.T) {
			users := &userRepoMock{
				getByEmailFn: func(context.Context, string) (service.User, error) {
					return service.User{}, service.ErrUserNotFound
				},
			}
			svc := newService(t, users, teamFound(), &tokenManagerMock{})

			gotID, err := svc.Register(context.Background(), service.RegisterParams{
				FullName:        testFullName,
				Email:           testEmail,
				Password:        testPassword,
				NationalTeamIDs: teamIDs,
			})

			require.NoError(t, err)
			require.NotEmpty(t, gotID)
			require.True(t, users.createCalled)
			require.Equal(t, teamIDs, users.createdUser.NationalTeamIDs)
		})
	}
}

// CT-039: with a single selection, the team existence check is performed with
// exactly the requested id.
func TestRegister_SingleSelection_ValidatesExactID(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	teams := teamFound()
	svc := newService(t, users, teams, &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID},
	})

	require.NoError(t, err)
	require.Equal(t, []string{testNationalTeamID}, teams.getIDCalls,
		"service must validate the exact requested selection id")
}

// CT-040: with three selections, the existence check runs once per id, in order.
func TestRegister_ThreeSelections_ValidatesEachID(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	teams := teamFound()
	svc := newService(t, users, teams, &tokenManagerMock{})

	teamIDs := []string{testNationalTeamID, testNationalTeamID2, testNationalTeamID3}
	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: teamIDs,
	})

	require.NoError(t, err)
	require.Equal(t, teamIDs, teams.getIDCalls,
		"service must validate each selection id exactly once, in order")
}

// CT-041: a single non-existent selection yields InvalidArgument and persists
// nothing.
func TestRegister_NonexistentSelection_InvalidArgument(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	teams := &nationalTeamRepoMock{
		getByIDFn: func(context.Context, string) (string, error) {
			return "", service.ErrNationalTeamNotFound
		},
	}
	svc := newService(t, users, teams, &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID},
	})

	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.False(t, users.createCalled, "must not persist when a selection is invalid")
}

// CT-042: when any position in the list is non-existent, Register returns
// InvalidArgument and persists nothing — regardless of which position fails.
func TestRegister_InvalidSelectionAtPosition_InvalidArgument(t *testing.T) {
	const badID = "99999999-9999-9999-9999-999999999999"
	cases := map[string][]string{
		"first":  {badID, testNationalTeamID, testNationalTeamID2},
		"second": {testNationalTeamID, badID, testNationalTeamID2},
		"third":  {testNationalTeamID, testNationalTeamID2, badID},
	}
	for name, teamIDs := range cases {
		t.Run(name, func(t *testing.T) {
			users := &userRepoMock{
				getByEmailFn: func(context.Context, string) (service.User, error) {
					return service.User{}, service.ErrUserNotFound
				},
			}
			teams := &nationalTeamRepoMock{
				getByIDFn: func(_ context.Context, id string) (string, error) {
					if id == badID {
						return "", service.ErrNationalTeamNotFound
					}
					return "Brasil", nil
				},
			}
			svc := newService(t, users, teams, &tokenManagerMock{})

			_, err := svc.Register(context.Background(), service.RegisterParams{
				FullName:        testFullName,
				Email:           testEmail,
				Password:        testPassword,
				NationalTeamIDs: teamIDs,
			})

			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.False(t, users.createCalled, "must not persist when any selection is invalid")
		})
	}
}

// CT-043: duplicate ids in the list yield InvalidArgument and persist nothing.
func TestRegister_DuplicateSelections_InvalidArgument(t *testing.T) {
	cases := map[string][]string{
		"adjacent": {testNationalTeamID, testNationalTeamID},
		"spread":   {testNationalTeamID, testNationalTeamID2, testNationalTeamID},
	}
	for name, teamIDs := range cases {
		t.Run(name, func(t *testing.T) {
			users := &userRepoMock{
				getByEmailFn: func(context.Context, string) (service.User, error) {
					return service.User{}, service.ErrUserNotFound
				},
			}
			svc := newService(t, users, teamFound(), &tokenManagerMock{})

			_, err := svc.Register(context.Background(), service.RegisterParams{
				FullName:        testFullName,
				Email:           testEmail,
				Password:        testPassword,
				NationalTeamIDs: teamIDs,
			})

			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.False(t, users.createCalled, "must not persist with duplicate selections")
		})
	}
}

// CT-045: GetUser returns the full list of the user's selections as populated by
// the repository.
func TestGetUser_ReturnsNationalTeamIDs(t *testing.T) {
	teamIDs := []string{testNationalTeamID, testNationalTeamID2}
	users := &userRepoMock{
		getByIDFn: func(_ context.Context, id string) (service.User, error) {
			return service.User{ID: id, Email: testEmail, NationalTeamIDs: teamIDs}, nil
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	got, err := svc.GetUser(context.Background(), testUserID)

	require.NoError(t, err)
	require.Equal(t, []string{testUserID}, users.getIDCalls, "service must forward the requested id")
	require.Equal(t, teamIDs, got.NationalTeamIDs)
}

// CT-046: GetUser with an unknown id maps the repository not-found sentinel to
// codes.NotFound.
func TestGetUser_UnknownID_NotFound(t *testing.T) {
	users := &userRepoMock{
		getByIDFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	_, err := svc.GetUser(context.Background(), "unknown-id")

	require.Equal(t, codes.NotFound, status.Code(err))
}

// CT-002: the persisted PasswordHash is never the plain password and is a valid
// bcrypt hash of it. Expected origin: re-derived via bcrypt (Gate 6 origin c),
// asserted on the argument captured from CreateUser, not on a mock return.
func TestRegister_PasswordNeverPlaintext(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID},
	})

	require.NoError(t, err)
	require.True(t, users.createCalled)
	require.NotEqual(t, testPassword, users.createdUser.PasswordHash, "password must not be stored in plaintext")
	// The captured hash must verify against the original plain password.
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(users.createdUser.PasswordHash), []byte(testPassword)),
		"stored hash must be a valid bcrypt hash of the plain password")
}

// CT-044: an already-used e-mail yields AlreadyExists and never persists, even
// with a valid list of selections.
// INVARIANT: Register on a taken e-mail returns AlreadyExists and calls neither
// CreateUser nor leaks which field collided.
func TestRegister_DuplicateEmail(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail}, nil
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID, testNationalTeamID2},
	})

	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.False(t, users.createCalled, "must not persist on duplicate e-mail")
}

// CT-006: a non-existent national team yields InvalidArgument and short-circuits
// before any e-mail lookup or persistence.
func TestRegister_NonexistentNationalTeam(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	teams := &nationalTeamRepoMock{
		getByIDFn: func(context.Context, string) (string, error) {
			return "", service.ErrNationalTeamNotFound
		},
	}
	svc := newService(t, users, teams, &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{"00000000-0000-4000-8000-000000000000"},
	})

	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, users.getEmailCalls, "must not check e-mail when team is invalid")
	require.False(t, users.createCalled)
}

// CT-016: when a user with the e-mail exists, the repository value flows back
// into the duplicate-detection path (interface consumed, e-mail forwarded).
// INVARIANT: GetUserByEmail is called with exactly the requested e-mail and a
// found user blocks registration.
func TestRegister_GetUserByEmail_Found(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(_ context.Context, email string) (service.User, error) {
			return service.User{ID: testUserID, Email: email}, nil
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	_, err := svc.Register(context.Background(), service.RegisterParams{
		FullName:        testFullName,
		Email:           testEmail,
		Password:        testPassword,
		NationalTeamIDs: []string{testNationalTeamID},
	})

	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.Equal(t, []string{testEmail}, users.getEmailCalls, "service must forward the requested e-mail unchanged")
}

// --- Login ----------------------------------------------------------------

// CT-007: correct credentials emit a token for the looked-up user id and return
// the TokenManager's expiration (now+1h with a fixed clock).
// INVARIANT: valid login issues a token for the persisted user id and surfaces
// the issuer's expiration.
func TestLogin_Success_JWT_TTL1h(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	expectedExp := now.Add(time.Hour)

	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail, PasswordHash: hashOf(t, testPassword)}, nil
		},
	}
	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) { return testToken, expectedExp, nil },
	}
	svc := newService(t, users, teamFound(), tokens)

	res, err := svc.Login(context.Background(), testEmail, testPassword)

	require.NoError(t, err)
	require.Equal(t, testToken, res.AccessToken)
	require.True(t, res.ExpiresAt.Equal(expectedExp), "expires_at must be the issuer's exp (now+1h)")
	require.False(t, res.PasswordChangeRequired, "a plain login without recovery must not require a password change")
	// Token was issued for the user resolved from the repository, not for the
	// raw input e-mail.
	require.Equal(t, testUserID, tokens.generatedFor)
}

// CT-008: a wrong password yields the generic Unauthenticated message and no
// token is issued.
func TestLogin_WrongPassword(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail, PasswordHash: hashOf(t, testPassword)}, nil
		},
	}
	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) {
			t.Fatal("Generate must not be called on wrong password")
			return "", time.Time{}, nil
		},
	}
	svc := newService(t, users, teamFound(), tokens)

	_, err := svc.Login(context.Background(), testEmail, "wrong-password")

	st, _ := status.FromError(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Equal(t, wrongPasswordMessage(t), st.Message())
	require.Empty(t, tokens.generatedFor)
}

// CT-009: a non-existent e-mail returns the EXACT same Unauthenticated message
// as a wrong password, AND still performs a bcrypt comparison (timing
// equalisation, RN5). The spy wraps the real bcrypt function so behaviour is
// unchanged; we only count invocations.
func TestLogin_NonexistentEmail_SameMessage(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	var compareCalls int
	restore := svc.SetCompareHash(func(hashedPassword, password []byte) error {
		compareCalls++
		return bcrypt.CompareHashAndPassword(hashedPassword, password)
	})
	defer restore()

	_, err := svc.Login(context.Background(), "ghost@example.com", testPassword)

	st, _ := status.FromError(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	// Anti-enumeration: identical message to the wrong-password path.
	require.Equal(t, wrongPasswordMessage(t), st.Message())
	// Anti-timing: bcrypt comparison ran despite the e-mail not existing.
	require.Equal(t, 1, compareCalls, "dummy bcrypt comparison must run on unknown e-mail")
}

// CT-017: a repository not-found on login is translated to Unauthenticated
// (never surfaced as the storage-level not-found error).
func TestLogin_GetUserByEmail_NotFound(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})

	_, err := svc.Login(context.Background(), "ghost@example.com", testPassword)

	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, errors.Is(err, service.ErrUserNotFound), "storage not-found must not leak to caller")
}

// wrongPasswordMessage runs the wrong-password login once and returns its
// message, so CT-009 asserts equality against an independently-produced value
// (Gate 6 origin c: re-derived by exercising the SUT) rather than a literal.
func wrongPasswordMessage(t *testing.T) string {
	t.Helper()
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail, PasswordHash: hashOf(t, testPassword)}, nil
		},
	}
	svc := newService(t, users, teamFound(), &tokenManagerMock{})
	_, err := svc.Login(context.Background(), testEmail, "definitely-wrong")
	st, _ := status.FromError(err)
	return st.Message()
}

// loginNow is the fixed clock instant used by the temporary-password login
// tests, so expiration boundaries are deterministic.
var loginNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// newLoginService builds an AuthService at loginNow for the temp-password login
// tests, wiring a TokenManager that issues testToken at now+1h.
func newLoginService(t *testing.T, user service.User) (*service.AuthService, *tokenManagerMock) {
	t.Helper()
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) { return user, nil },
	}
	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) { return testToken, loginNow.Add(time.Hour), nil },
	}
	svc, err := service.NewAuthService(users, teamFound(), tokens, &emailSenderMock{}, fixedClock{now: loginNow}, zap.NewNop(), testCost)
	require.NoError(t, err)
	return svc, tokens
}

const tempPassword = "TempXxx-RECOVERY-9!"

// CT-006: when the permanent password matches, access is granted with
// PasswordChangeRequired=false even though a non-expired temporary password is
// active — the permanent password is checked first (CA-10/RN10).
func TestLogin_SenhaOriginal_ComTempValida_False(t *testing.T) {
	user := service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, tempPassword),
		TempPasswordExpiresAt: loginNow.Add(5 * time.Minute),
	}
	svc, tokens := newLoginService(t, user)

	res, err := svc.Login(context.Background(), testEmail, testPassword)

	require.NoError(t, err)
	require.Equal(t, testToken, res.AccessToken)
	require.False(t, res.PasswordChangeRequired, "the permanent password must grant normal access despite an active temp password")
	require.Equal(t, testUserID, tokens.generatedFor)
}

// CT-007: an active, non-expired temporary password grants access and signals
// PasswordChangeRequired=true (CA-04).
func TestLogin_SenhaTemporariaValida_True(t *testing.T) {
	user := service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, tempPassword),
		TempPasswordExpiresAt: loginNow.Add(10 * time.Minute),
	}
	svc, tokens := newLoginService(t, user)

	res, err := svc.Login(context.Background(), testEmail, tempPassword)

	require.NoError(t, err)
	require.Equal(t, testToken, res.AccessToken)
	require.True(t, res.PasswordChangeRequired, "an active temp password must require a password change")
	require.Equal(t, testUserID, tokens.generatedFor)
}

// CT-008: an expired temporary password (expires_at = now-1s) is rejected with
// the generic Unauthenticated message and issues no token (CA-05).
func TestLogin_TempExpirada_Unauthenticated(t *testing.T) {
	user := service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, tempPassword),
		TempPasswordExpiresAt: loginNow.Add(-time.Second),
	}
	svc, tokens := newLoginService(t, user)

	_, err := svc.Login(context.Background(), testEmail, tempPassword)

	st, _ := status.FromError(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Equal(t, wrongPasswordMessage(t), st.Message())
	require.Empty(t, tokens.generatedFor, "no token must be issued for an expired temp password")
}

// CT-009: the expiration boundary is exclusive — expires_at == clock.Now() is
// treated as expired (Before, not Before||Equal), so a temp password whose
// expiration is exactly now is rejected (CA-05).
func TestLogin_FronteiraExpiracaoExata_Negado(t *testing.T) {
	user := service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, tempPassword),
		TempPasswordExpiresAt: loginNow, // expires_at == clock.Now()
	}
	svc, tokens := newLoginService(t, user)

	_, err := svc.Login(context.Background(), testEmail, tempPassword)

	require.Equal(t, codes.Unauthenticated, status.Code(err),
		"expires_at == now must be denied (boundary is exclusive)")
	require.Empty(t, tokens.generatedFor)
}

// CT-010: the permanent password grants normal access with
// PasswordChangeRequired=false even when the temporary password has already
// expired (CA-10).
func TestLogin_SenhaOriginal_TempExpirada_False(t *testing.T) {
	user := service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, tempPassword),
		TempPasswordExpiresAt: loginNow.Add(-time.Hour),
	}
	svc, tokens := newLoginService(t, user)

	res, err := svc.Login(context.Background(), testEmail, testPassword)

	require.NoError(t, err)
	require.Equal(t, testToken, res.AccessToken)
	require.False(t, res.PasswordChangeRequired)
	require.Equal(t, testUserID, tokens.generatedFor)
}

// CT-017 (task T6): a user with no pending recovery logs in normally with
// PasswordChangeRequired=false (CA-08).
func TestLogin_SemRecovery_False(t *testing.T) {
	user := service.User{
		ID:           testUserID,
		Email:        testEmail,
		PasswordHash: hashOf(t, testPassword),
		// TempPasswordHash empty, TempPasswordExpiresAt zero.
	}
	svc, tokens := newLoginService(t, user)

	res, err := svc.Login(context.Background(), testEmail, testPassword)

	require.NoError(t, err)
	require.Equal(t, testToken, res.AccessToken)
	require.False(t, res.PasswordChangeRequired)
	require.Equal(t, testUserID, tokens.generatedFor)
}

// --- Password recovery ----------------------------------------------------

const knownTempPassword = "TempPass-KNOWN-1!"

// newRecoveryService builds an AuthService with an injected fixed clock, email
// sender and temp-password generator, returning all three so tests can assert
// the side-effects. The generator is overridden to a known value via the seam.
func newRecoveryService(
	t *testing.T,
	users service.UserRepository,
	email *emailSenderMock,
	now time.Time,
	tempPassword string,
) *service.AuthService {
	t.Helper()
	svc, err := service.NewAuthService(users, teamFound(), &tokenManagerMock{}, email, fixedClock{now: now}, zap.NewNop(), testCost)
	require.NoError(t, err)
	svc.SetGenTempPassword(func() (string, error) { return tempPassword, nil })
	return svc
}

// CT-001: a registered e-mail persists a bcrypt-valid hash of the (known)
// temporary password with expires_at = now+15min and sends exactly one e-mail.
// The stored hash is verified by re-deriving against the known plain password,
// not by trusting any mock-planted value.
func TestProcessRecovery_EmailCadastrado_PersisteEEnvia(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail}, nil
		},
	}
	email := &emailSenderMock{}
	svc := newRecoveryService(t, users, email, now, knownTempPassword)

	err := svc.ProcessRecovery(context.Background(), testEmail)

	require.NoError(t, err)
	require.True(t, users.setTempCalled, "SetTempPassword must be called")
	require.Equal(t, testUserID, users.setTempID)
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(users.setTempHash), []byte(knownTempPassword)),
		"persisted hash must be a valid bcrypt hash of the temporary password")
	require.True(t, users.setTempExpiresAt.Equal(now.Add(15*time.Minute)),
		"expires_at must be clock.Now()+15min")
	require.True(t, email.sendCalled, "exactly one e-mail must be sent")
	require.Equal(t, testEmail, email.toArg)
	require.NotContains(t, email.subjectArg, knownTempPassword, "subject must not carry the password")
}

// CT-002: an unknown e-mail produces total silence — no persistence, no e-mail,
// and a nil error (anti-enumeration, CA-02).
func TestProcessRecovery_EmailNaoCadastrado_Silencio(t *testing.T) {
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{}, service.ErrUserNotFound
		},
	}
	email := &emailSenderMock{}
	svc := newRecoveryService(t, users, email, time.Now(), knownTempPassword)

	err := svc.ProcessRecovery(context.Background(), "ghost@example.com")

	require.NoError(t, err)
	require.False(t, users.setTempCalled, "must not persist a temp password for an unknown e-mail")
	require.False(t, email.sendCalled, "must not send an e-mail for an unknown e-mail")
}

// CT-003: a delivery failure is best-effort — the temp password is already
// persisted, processRecovery still returns nil and the flow is unchanged
// (RN4/CA-03). The error is logged without the password (verified by routing
// through a nop logger; no password leaves the body argument).
func TestProcessRecovery_FalhaEnvio_BestEffort(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	users := &userRepoMock{
		getByEmailFn: func(context.Context, string) (service.User, error) {
			return service.User{ID: testUserID, Email: testEmail}, nil
		},
	}
	email := &emailSenderMock{err: errors.New("resend timeout")}
	svc := newRecoveryService(t, users, email, now, knownTempPassword)

	err := svc.ProcessRecovery(context.Background(), testEmail)

	require.NoError(t, err, "delivery failure must not surface as an error")
	require.True(t, users.setTempCalled, "temp password must remain persisted despite send failure")
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(users.setTempHash), []byte(knownTempPassword)),
		"persisted hash must be valid even when delivery fails")
	require.True(t, email.sendCalled, "delivery must have been attempted")
}

// --- ChangePassword -------------------------------------------------------

// changeNow is the fixed instant used by the ChangePassword unit tests so the
// expiration boundary is deterministic.
var changeNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

const (
	changeTempPassword = "TempXxx-RECOVERY-9!"
	changeNewPassword  = "NovaSenha@2026!"
)

// newChangeService builds an AuthService at changeNow whose GetUserByID returns
// the given user, exposing the email mock so tests can assert the notification
// side-effects. The bcrypt comparator is the real one (MinCost), so temp
// validation is exercised end-to-end rather than mock-driven.
func newChangeService(t *testing.T, user service.User, email *emailSenderMock) (*service.AuthService, *userRepoMock) {
	t.Helper()
	users := &userRepoMock{
		getByIDFn: func(context.Context, string) (service.User, error) { return user, nil },
	}
	svc, err := service.NewAuthService(users, teamFound(), &tokenManagerMock{}, email, fixedClock{now: changeNow}, zap.NewNop(), testCost)
	require.NoError(t, err)
	return svc, users
}

// activeTempUser returns a user carrying an active (non-expired) temporary
// password whose plain value is changeTempPassword.
func activeTempUser(t *testing.T) service.User {
	t.Helper()
	return service.User{
		ID:                    testUserID,
		Email:                 testEmail,
		PasswordHash:          hashOf(t, testPassword),
		TempPasswordHash:      hashOf(t, changeTempPassword),
		TempPasswordExpiresAt: changeNow.Add(10 * time.Minute),
	}
}

// CT-011: a correct, non-expired temporary password persists the new password
// via UpdatePassword. The stored hash is verified by re-deriving against the new
// plain password (not a mock-planted value), proving the temp was invalidated in
// the same operation (UpdatePassword clears the temp columns).
func TestChangePassword_TempValida_PersisteEInvalida(t *testing.T) {
	svc, users := newChangeService(t, activeTempUser(t), &emailSenderMock{})

	err := svc.ChangePassword(context.Background(), testUserID, changeTempPassword, changeNewPassword)

	require.NoError(t, err)
	require.True(t, users.updateCalled, "UpdatePassword must be called exactly once")
	require.Equal(t, testUserID, users.updateID, "must update the user identified by the token sub")
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(users.updateHash), []byte(changeNewPassword)),
		"persisted hash must be a valid bcrypt hash of the new password")
	require.NotEqual(t, changeNewPassword, users.updateHash, "new password must not be stored in plaintext")
}

// CT-012: an incorrect temporary password is rejected with InvalidArgument and
// nothing is persisted.
func TestChangePassword_TempIncorreta_InvalidArgument(t *testing.T) {
	svc, users := newChangeService(t, activeTempUser(t), &emailSenderMock{})

	err := svc.ChangePassword(context.Background(), testUserID, "SenhaErrada!", changeNewPassword)

	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.False(t, users.updateCalled, "must not persist when the temp password is wrong")
}

// CT-013: a correct but expired temporary password is rejected even though the
// hash matches — the Before boundary treats expires_at <= now as expired.
func TestChangePassword_TempExpirada_InvalidArgument(t *testing.T) {
	user := activeTempUser(t)
	user.TempPasswordExpiresAt = changeNow.Add(-time.Minute)
	svc, users := newChangeService(t, user, &emailSenderMock{})

	err := svc.ChangePassword(context.Background(), testUserID, changeTempPassword, changeNewPassword)

	st, _ := status.FromError(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "senha temporária inválida ou expirada", st.Message())
	require.False(t, users.updateCalled, "must not persist when the temp password is expired")
}

// CT-014: a user with no active recovery (empty temp hash, zero expiration) is
// rejected — ChangePassword only works post-recovery, never with the current
// password.
func TestChangePassword_SemTempAtiva_InvalidArgument(t *testing.T) {
	user := service.User{
		ID:           testUserID,
		Email:        testEmail,
		PasswordHash: hashOf(t, testPassword),
		// TempPasswordHash empty, TempPasswordExpiresAt zero.
	}
	svc, users := newChangeService(t, user, &emailSenderMock{})

	err := svc.ChangePassword(context.Background(), testUserID, "qualquer", changeNewPassword)

	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.False(t, users.updateCalled, "must not persist when no recovery is pending")
}

// CT-015: a successful change sends the notification e-mail after persisting,
// addressed to the user, and the body carries no password.
func TestChangePassword_DisparaNotificacao(t *testing.T) {
	email := &emailSenderMock{}
	svc, users := newChangeService(t, activeTempUser(t), email)

	err := svc.ChangePassword(context.Background(), testUserID, changeTempPassword, changeNewPassword)

	require.NoError(t, err)
	require.True(t, users.updateCalled, "the change must be persisted")
	require.True(t, email.sendCalled, "a notification e-mail must be sent after the change")
	require.Equal(t, testEmail, email.toArg)
	require.NotContains(t, email.bodyArg, changeNewPassword, "body must not carry the new password")
	require.NotContains(t, email.bodyArg, changeTempPassword, "body must not carry the temp password")
}

// CT-016: a notification delivery failure is best-effort — UpdatePassword ran
// before Send, so the change stays persisted and ChangePassword returns nil.
func TestChangePassword_FalhaNotificacao_TrocaEfetivada(t *testing.T) {
	email := &emailSenderMock{err: errors.New("smtp timeout")}
	svc, users := newChangeService(t, activeTempUser(t), email)

	err := svc.ChangePassword(context.Background(), testUserID, changeTempPassword, changeNewPassword)

	require.NoError(t, err, "a delivery failure must not surface as an error")
	require.True(t, users.updateCalled, "the change must be persisted before the e-mail is attempted")
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(users.updateHash), []byte(changeNewPassword)),
		"the persisted hash must remain valid despite the delivery failure")
	require.True(t, email.sendCalled, "delivery must have been attempted")
}

// CT-036: neither the temporary password nor the new password may appear in the
// notification subject or body.
func TestChangePassword_NotificacaoNaoVazaSenha(t *testing.T) {
	email := &emailSenderMock{}
	svc, _ := newChangeService(t, activeTempUser(t), email)

	err := svc.ChangePassword(context.Background(), testUserID, changeTempPassword, changeNewPassword)

	require.NoError(t, err)
	require.True(t, email.sendCalled)
	require.NotContains(t, email.subjectArg, changeTempPassword, "subject must not carry the temp password")
	require.NotContains(t, email.subjectArg, changeNewPassword, "subject must not carry the new password")
	require.NotContains(t, email.bodyArg, changeTempPassword, "body must not carry the temp password")
	require.NotContains(t, email.bodyArg, changeNewPassword, "body must not carry the new password")
}

// CT-037: the production crypto/rand generator yields distinct strings of the
// expected length on successive calls (no overridden seam).
func TestGenTempPassword_CryptoRand_Unicidade(t *testing.T) {
	const minLength = 12

	first, err := service.GenerateTempPassword()
	require.NoError(t, err)
	second, err := service.GenerateTempPassword()
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(first), minLength, "generated password must meet the minimum length")
	require.GreaterOrEqual(t, len(second), minLength)
	require.NotEqual(t, first, second, "successive generations must differ")
}
