// Package email provides concrete implementations of the service.EmailSender
// interface for transactional e-mail delivery.
package email

import (
	"context"

	"go.uber.org/zap"
)

// NoopSender is a dev/test sender that logs the destination and subject at
// info level and discards the message body. The body MUST NOT appear in any
// log entry (CA-09: the body may contain a temporary password).
type NoopSender struct {
	logger *zap.Logger
}

// NewNoopSender constructs a NoopSender backed by the given zap logger.
func NewNoopSender(logger *zap.Logger) *NoopSender {
	return &NoopSender{logger: logger}
}

// Send logs to/subject at info level and returns nil. The body is never
// emitted so that credentials embedded in the body are not leaked (CA-09).
func (s *NoopSender) Send(_ context.Context, to, subject, _ string) error {
	s.logger.Info("email: send (noop)",
		zap.String("to", to),
		zap.String("subject", subject),
	)
	return nil
}
