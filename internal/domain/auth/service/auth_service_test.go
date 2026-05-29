package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	getByEmailFn func(ctx context.Context, email string) (service.User, error)
	getByIDFn    func(ctx context.Context, id string) (service.User, error)
	createFn     func(ctx context.Context, u service.User) error

	createdUser   service.User
	createCalled  bool
	getEmailCalls []string
	getIDCalls    []string
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
	svc, err := service.NewAuthService(users, teams, tokens, fixedClock{now: time.Now()}, testCost)
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
