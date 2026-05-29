// Package token implements issuing and validating JWT HS256 access tokens.
//
// The TokenManager signs tokens with an injected HMAC-SHA256 secret and a
// configurable TTL, and validates them with strict algorithm enforcement
// (HS256 only) as a defense against alg-confusion attacks (none / RS256).
// All time reads go through an injected clock.Clock for deterministic tests.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
)

// ErrEmptyUserID is returned by Generate when the userID (sub claim) is empty.
var ErrEmptyUserID = errors.New("user id must not be empty")

// signingMethod is the single algorithm this manager issues and accepts.
const signingMethod = "HS256"

// TokenManager issues and validates JWT HS256 access tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	clk    clock.Clock
}

// NewTokenManager constructs a TokenManager with the given HMAC-SHA256 secret,
// token TTL and clock.
func NewTokenManager(secret []byte, ttl time.Duration, clk clock.Clock) *TokenManager {
	return &TokenManager{secret: secret, ttl: ttl, clk: clk}
}

// Generate issues a signed HS256 token whose sub claim is userID and whose exp
// claim is clk.Now()+ttl. It returns the signed token and its expiration time.
func (m *TokenManager) Generate(userID string) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, ErrEmptyUserID
	}

	now := m.clk.Now()
	expiresAt := now.Add(m.ttl)

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}

	return signed, expiresAt, nil
}

// Validate verifies the token signature, algorithm (HS256 only) and expiration
// (using clk.Now()), returning the sub claim on success. Errors wrap the
// golang-jwt sentinels (jwt.ErrTokenExpired, jwt.ErrTokenSignatureInvalid, ...)
// so callers can branch with errors.Is.
func (m *TokenManager) Validate(tokenString string) (string, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{signingMethod}),
		jwt.WithTimeFunc(m.clk.Now),
	)

	claims := &jwt.RegisteredClaims{}
	_, err := parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return "", err
	}

	return claims.Subject, nil
}
