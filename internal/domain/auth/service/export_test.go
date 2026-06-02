package service

import "context"

// ProcessRecovery exposes the synchronous core of the recovery flow to the
// external test package, so the goroutine-driven RequestPasswordRecovery can be
// exercised deterministically without depending on goroutine scheduling.
func (s *AuthService) ProcessRecovery(ctx context.Context, email string) error {
	return s.processRecovery(ctx, email)
}

// SetGenTempPassword replaces the temporary-password generator for tests,
// returning a restore func. The seam exists in production (crypto/rand is wired
// in NewAuthService); this only opens it so a known value can be injected.
func (s *AuthService) SetGenTempPassword(fn func() (string, error)) (restore func()) {
	prev := s.genTempPassword
	s.genTempPassword = fn
	return func() { s.genTempPassword = prev }
}

// GenerateTempPassword exposes the production crypto/rand generator so its
// uniqueness and length can be asserted without constructing a service.
func GenerateTempPassword() (string, error) {
	return generateTempPassword()
}

// SetCompareHash replaces the bcrypt comparison function for tests, returning a
// restore func. It exposes the injected comparator seam to the external
// service_test package so timing-equalisation behaviour (the dummy comparison
// on the unknown-e-mail login path) can be observed with a spy. The seam exists
// in production (bcrypt is wired in NewAuthService); this only opens it to tests.
func (s *AuthService) SetCompareHash(fn func(hashedPassword, password []byte) error) (restore func()) {
	prev := s.compareHash
	s.compareHash = fn
	return func() { s.compareHash = prev }
}
