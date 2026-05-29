package token_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
)

const (
	testSecret = "test-secret-at-least-32-bytes-long-xxxx"
	testTTL    = time.Hour
	testUserID = "11111111-2222-3333-4444-555555555555"
)

// fixedClock is a deterministic Clock returning a controllable time.
type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time { return c.now }

func (c *fixedClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func newManagerAt(now time.Time) (*token.TokenManager, *fixedClock) {
	clk := &fixedClock{now: now}
	return token.NewTokenManager([]byte(testSecret), testTTL, clk), clk
}

// CT-F11: Generate emits a parseable HS256 token with sub and exp = now+TTL.
func TestGenerate_ClaimsAndExp(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tm, _ := newManagerAt(now)

	tokenStr, expiresAt, err := tm.Generate(testUserID)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)
	require.True(t, expiresAt.Equal(now.Add(testTTL)), "expiresAt must be now+TTL")

	// Parse independently to assert the real on-the-wire claims and algorithm,
	// not values the SUT could have fabricated. The deterministic clock is fed to
	// the parser too (WithTimeFunc) so expiry validation anchors on the test's
	// fixed `now`, never on the real wall clock.
	parsed, err := jwt.Parse(tokenStr, func(*jwt.Token) (any, error) {
		return []byte(testSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(func() time.Time { return now }))
	require.NoError(t, err)
	require.Equal(t, "HS256", parsed.Method.Alg())

	sub, err := parsed.Claims.GetSubject()
	require.NoError(t, err)
	require.Equal(t, testUserID, sub)

	exp, err := parsed.Claims.GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, now.Add(testTTL).Unix(), exp.Unix())
}

// CT-F12: a freshly issued token is accepted and returns the userID.
func TestValidate_FreshToken(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tm, _ := newManagerAt(now)

	tokenStr, _, err := tm.Generate(testUserID)
	require.NoError(t, err)

	gotUserID, err := tm.Validate(tokenStr)
	require.NoError(t, err)
	require.Equal(t, testUserID, gotUserID)
}

// CT-F13: a token expires once the clock advances past the TTL.
func TestValidate_ExpiredToken(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tm, clk := newManagerAt(now)

	tokenStr, _, err := tm.Generate(testUserID)
	require.NoError(t, err)

	clk.advance(testTTL + time.Minute)

	_, err = tm.Validate(tokenStr)
	require.Error(t, err)
	require.ErrorIs(t, err, jwt.ErrTokenExpired)
}

// CT-F14: a token signed with a different secret is rejected.
func TestValidate_WrongSecret(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	issuer, _ := newManagerAt(now)

	tokenStr, _, err := issuer.Generate(testUserID)
	require.NoError(t, err)

	// Validator configured with a different secret.
	validator := token.NewTokenManager([]byte("another-secret-at-least-32-bytes-long"), testTTL, &fixedClock{now: now})

	_, err = validator.Validate(tokenStr)
	require.Error(t, err)
	require.ErrorIs(t, err, jwt.ErrTokenSignatureInvalid)
}

// CT-F15 / CT-F16: alg-confusion defense — tokens crafted with disallowed
// algorithms (none, RS256) are rejected by the HS256-only validator.
func TestValidate_DisallowedAlgorithms_Rejected(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tm, _ := newManagerAt(now)

	tests := []struct {
		name  string
		token string
	}{
		{name: "alg_none", token: craftNoneToken(t, testUserID, now.Add(testTTL))},
		{name: "alg_RS256", token: craftRS256Token(t, testUserID, now.Add(testTTL))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tm.Validate(tc.token)
			require.Error(t, err)
			// WithValidMethods rejects the algorithm before key verification,
			// wrapping ErrTokenSignatureInvalid.
			require.ErrorIs(t, err, jwt.ErrTokenSignatureInvalid)
		})
	}
}

// CT-F17: an empty userID must not be accepted silently.
func TestGenerate_EmptyUserID(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tm, _ := newManagerAt(now)

	_, _, err := tm.Generate("")
	require.Error(t, err)
	require.ErrorIs(t, err, token.ErrEmptyUserID)
}

// --- helpers: manually crafted tokens (NOT via Generate) ---

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
	// none tokens carry an empty signature segment.
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

// guard against accidental reliance on errors.New string matching.
var _ = errors.Is
