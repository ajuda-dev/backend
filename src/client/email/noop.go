package email

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"go.uber.org/zap"
)

type noopSender struct{}

func NewNoopSender() EmailSender {
	return &noopSender{}
}

func (n *noopSender) Send(to, subject, body string) error {
	logger.Info("email skipped (noop)",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("body", body))
	return nil
}
