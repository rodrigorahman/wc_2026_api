package service

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
