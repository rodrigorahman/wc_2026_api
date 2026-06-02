package email

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	resendgo "github.com/resend/resend-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
)

// Compile-time interface assertions (CT-T3-004, CT-T3-005).
var _ service.EmailSender = (*NoopSender)(nil)
var _ service.EmailSender = (*ResendSender)(nil)

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func newObservedLogger(t *testing.T) (*zap.Logger, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zap.InfoLevel)
	return zap.New(core), logs
}

// newResendSenderWithClient constructs a ResendSender with an injected client,
// allowing tests to substitute the real Resend SDK client with a fake.
func newResendSenderWithClient(client resendClient, fromEmail string, logger *zap.Logger) *ResendSender {
	return &ResendSender{
		client:    client,
		fromEmail: fromEmail,
		logger:    logger,
	}
}

// captureClient is a resendClient stub that records the last request and
// returns a configurable error.
type captureClient struct {
	captured *resendgo.SendEmailRequest
	err      error
}

func (c *captureClient) SendWithContext(_ context.Context, req *resendgo.SendEmailRequest) (*resendgo.SendEmailResponse, error) {
	c.captured = req
	if c.err != nil {
		return nil, c.err
	}
	return &resendgo.SendEmailResponse{Id: "test-id"}, nil
}

// Ensure captureClient satisfies the local interface so tests remain
// package-internal and never expose the interface outside infra/email.
var _ resendClient = (*captureClient)(nil)

// ----------------------------------------------------------------------------
// NoopSender tests
// ----------------------------------------------------------------------------

// CT-T3-001: Send returns nil and the body never appears in any log entry.
func TestNoopSender_Send_RetornaNilENaoLogaBody(t *testing.T) {
	logger, logs := newObservedLogger(t)
	s := NewNoopSender(logger)

	const body = "SUA_SENHA_TEMPORARIA_S3cr3t123"
	err := s.Send(context.Background(), "to@example.com", "subject", body)

	require.NoError(t, err)
	require.GreaterOrEqual(t, logs.Len(), 1)
	for _, entry := range logs.All() {
		assert.NotContains(t, entry.Message, body, "body must not appear in log message")
		for _, f := range entry.Context {
			assert.NotEqual(t, body, f.String, "body must not appear in any log field")
		}
	}
}

// CT-T3-002: Par negativo de CT-T3-001 — body contendo senha temporária
// nunca vaza em log (segurança autônoma; CA-09).
func TestNoopSender_Send_NaoVazaBodyComSenha(t *testing.T) {
	logger, logs := newObservedLogger(t)
	s := NewNoopSender(logger)

	const body = "SENHA_TEMPORARIA_SECRETA_XYZ"
	_ = s.Send(context.Background(), "user@test.com", "Recuperação de acesso", body)

	for _, entry := range logs.All() {
		require.NotContains(t, entry.Message, body)
		for _, f := range entry.Context {
			require.NotEqual(t, body, f.String)
		}
	}
}

// CT-T3-003: Send logs to and subject at info level; body absent from every entry.
func TestNoopSender_Send_LogaToESubjectInfo(t *testing.T) {
	logger, logs := newObservedLogger(t)
	s := NewNoopSender(logger)

	const (
		to      = "dev@example.com"
		subject = "Recuperação de acesso — Bolão Copa 2026"
		body    = "irrelevant-body"
	)

	err := s.Send(context.Background(), to, subject, body)
	require.NoError(t, err)

	require.GreaterOrEqual(t, logs.Len(), 1, "expected at least one info log entry")

	found := false
	for _, entry := range logs.All() {
		hasTo, hasSubject := false, false
		for _, f := range entry.Context {
			if f.Key == "to" && f.String == to {
				hasTo = true
			}
			if f.Key == "subject" && f.String == subject {
				hasSubject = true
			}
		}
		if hasTo && hasSubject {
			found = true
		}
		// body must never appear
		assert.NotContains(t, entry.Message, body)
		for _, f := range entry.Context {
			assert.NotEqual(t, body, f.String)
		}
	}
	require.True(t, found, "expected a log entry containing both 'to' and 'subject' fields")
}

// ----------------------------------------------------------------------------
// ResendSender unit tests (mock resendClient)
// ----------------------------------------------------------------------------

// CT-T3-006: SDK error is propagated to the caller unchanged (errors.Is).
func TestResendSender_Send_PropagaErroSDK(t *testing.T) {
	logger, _ := newObservedLogger(t)
	sdkErr := errors.New("resend: rate limit")
	stub := &captureClient{err: sdkErr}
	s := newResendSenderWithClient(stub, "from@example.com", logger)

	err := s.Send(context.Background(), "to@example.com", "subject", "body")
	require.Error(t, err)
	require.True(t, errors.Is(err, sdkErr), "expected propagated SDK error via errors.Is")
}

// CT-T3-007: happy path — no error from SDK → Send returns nil.
func TestResendSender_Send_SucessoRetornaNil(t *testing.T) {
	logger, _ := newObservedLogger(t)
	stub := &captureClient{}
	s := newResendSenderWithClient(stub, "from@example.com", logger)

	err := s.Send(context.Background(), "player@example.com", "Acesso recuperado", "body text")
	require.NoError(t, err)
}

// CT-T3-008: the request built by Send has the correct From, To, and Subject
// (asserted against captured fields, not planted values).
func TestResendSender_Send_MontaRequestFromToSubject(t *testing.T) {
	const (
		fromEmail = "noreply@wc2026.example.com"
		toEmail   = "player@example.com"
		subject   = "Acesso recuperado"
		body      = "plain text body"
	)

	logger, _ := newObservedLogger(t)
	stub := &captureClient{}
	s := newResendSenderWithClient(stub, fromEmail, logger)

	err := s.Send(context.Background(), toEmail, subject, body)
	require.NoError(t, err)
	require.NotNil(t, stub.captured, "expected the SDK client to be called")

	assert.Equal(t, fromEmail, stub.captured.From)
	assert.Equal(t, []string{toEmail}, stub.captured.To)
	assert.Equal(t, subject, stub.captured.Subject)
	assert.Equal(t, body, stub.captured.Text)
}

// ----------------------------------------------------------------------------
// CT-T3-009: integration — real HTTP boundary via httptest, 401 propagates.
// ----------------------------------------------------------------------------

func TestResendSender_Send_HTTPReal_ErroAutenticacao(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)

	resendClient := resendgo.NewCustomClient(srv.Client(), "invalid-key")
	resendClient.BaseURL = baseURL

	s := newResendSenderWithClient(resendClient.Emails, "from@example.com", zap.NewNop())

	sendErr := s.Send(context.Background(), "player@example.com", "Test", "body")
	require.Error(t, sendErr, "expected an error for 401 Unauthorized")
	assert.ErrorContains(t, sendErr, "[ERROR]: unauthorized",
		"expected the SDK error message surfaced from the real HTTP boundary")
}
