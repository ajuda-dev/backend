package email

import (
	"os"
	"strings"

	"github.com/ajuda-dev/backend/src/client/email/brevoimpl"
	"github.com/ajuda-dev/backend/src/config/logger"
	"go.uber.org/zap"
)

var (
	ProviderEnv  = "EMAIL_PROVIDER"
	BrevoHostEnv = "BREVO_SMTP_HOST"
	BrevoPortEnv = "BREVO_SMTP_PORT"
	BrevoUserEnv = "BREVO_SMTP_USER"
	BrevoKeyEnv  = "BREVO_SMTP_KEY"
	BrevoFromEnv = "BREVO_FROM"
)

const ProviderBrevo = "brevo"

// FromEnv escolhe a implementação pelo EMAIL_PROVIDER (default brevo).
// Credenciais vazias viram noop: sem rede e sem erro, para CI e dev local.
func FromEnv() EmailSender {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv(ProviderEnv)))
	if provider == "" {
		provider = ProviderBrevo
	}

	switch provider {
	case ProviderBrevo:
		return brevoFromEnv()
	default:
		logger.Info("email provider unknown, using noop",
			zap.String("provider", provider))
		return NewNoopSender()
	}
}

func brevoFromEnv() EmailSender {
	user := strings.TrimSpace(os.Getenv(BrevoUserEnv))
	key := strings.TrimSpace(os.Getenv(BrevoKeyEnv))
	from := strings.TrimSpace(os.Getenv(BrevoFromEnv))
	if user == "" || key == "" || from == "" {
		logger.Info("email sender disabled",
			zap.String("provider", ProviderBrevo),
			zap.String("reason", BrevoUserEnv+", "+BrevoKeyEnv+" and "+BrevoFromEnv+" are not configured"))
		return NewNoopSender()
	}

	return brevoimpl.NewSender(
		os.Getenv(BrevoHostEnv),
		os.Getenv(BrevoPortEnv),
		user,
		key,
		from,
	)
}
