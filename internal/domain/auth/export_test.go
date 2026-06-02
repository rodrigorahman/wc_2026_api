package auth

import (
	"go.uber.org/zap"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
)

// ProvideEmailSender exposes the unexported provideEmailSender for wiring tests
// that assert the config-driven selection of NoopSender vs ResendSender.
func ProvideEmailSender(cfg config.Config, logger *zap.Logger) service.EmailSender {
	return provideEmailSender(cfg, logger)
}

// NewUserRepositoryAdapter exposes the unexported userRepositoryAdapter (bound
// to the service.UserRepository interface) for tests that assert temp-field
// propagation and sentinel translation through the real concrete repository.
func NewUserRepositoryAdapter(repo *repository.UserRepository) service.UserRepository {
	return userRepositoryAdapter{repo: repo}
}
