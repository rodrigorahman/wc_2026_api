package email

import (
	"context"

	resendgo "github.com/resend/resend-go/v2"
	"go.uber.org/zap"
)

// resendClient is the minimal interface over the Resend SDK required by
// ResendSender. Declaring it here (in the consumer) enables unit tests to
// inject a capturing fake without hitting real HTTP.
type resendClient interface {
	SendWithContext(ctx context.Context, params *resendgo.SendEmailRequest) (*resendgo.SendEmailResponse, error)
}

// ResendSender delivers transactional e-mail via the Resend API.
type ResendSender struct {
	client    resendClient
	fromEmail string
	logger    *zap.Logger
}

// NewResendSender constructs a production ResendSender that uses the official
// Resend SDK client under the hood.
func NewResendSender(apiKey, fromEmail string, logger *zap.Logger) *ResendSender {
	c := resendgo.NewClient(apiKey)
	return &ResendSender{
		client:    c.Emails,
		fromEmail: fromEmail,
		logger:    logger,
	}
}

// Send delivers a plain-text e-mail. It propagates the SDK error unchanged so
// the caller can inspect it with errors.Is.
func (s *ResendSender) Send(ctx context.Context, to, subject, body string) error {
	req := &resendgo.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	}
	_, err := s.client.SendWithContext(ctx, req)
	return err
}
