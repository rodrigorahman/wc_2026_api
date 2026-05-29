// Package clock provides an injectable time source so that time-dependent
// logic (JWT expiration, etc.) can be made deterministic in tests.
package clock

import "time"

// Clock abstracts the current time. Production code uses the system clock;
// tests inject a fixed or advanceable clock for deterministic behaviour.
type Clock interface {
	Now() time.Time
}

// systemClock is the production Clock backed by the operating-system time.
type systemClock struct{}

// New returns a Clock backed by the system wall clock.
func New() Clock {
	return systemClock{}
}

// Now returns the current system time.
func (systemClock) Now() time.Time {
	return time.Now()
}
