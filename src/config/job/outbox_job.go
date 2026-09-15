package job

import (
	"os"
	"strconv"
	"time"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OutboxConfig struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
}

type OutboxHandler interface {
	Handle(event domain.OutboxEventDomain) error
}

type LogOutboxHandler struct{}

func (LogOutboxHandler) Handle(event domain.OutboxEventDomain) error {
	logger.Info("outbox event processed",
		zap.Int64("id", event.Id),
		zap.String("type", event.Type),
		zap.String("user_id", event.UserId))
	return nil
}

func OutboxConfigFromEnv() OutboxConfig {
	cfg := OutboxConfig{
		Enabled:      true,
		PollInterval: 2 * time.Second,
		BatchSize:    50,
	}
	if v := os.Getenv("OUTBOX_ENABLED"); v != "" {
		cfg.Enabled = v != "false" && v != "0"
	}
	if v := os.Getenv("OUTBOX_POLL_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.PollInterval = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv("OUTBOX_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BatchSize = n
		}
	}
	return cfg
}

func StartOutboxJob(db *gorm.DB, cfg OutboxConfig, handler OutboxHandler) {
	if !cfg.Enabled {
		logger.Info("outbox job disabled")
		return
	}
	if handler == nil {
		handler = LogOutboxHandler{}
	}
	repo := repository.NewOutboxEventRepository(db)
	ticker := time.NewTicker(cfg.PollInterval)
	go func() {
		for range ticker.C {
			RunOutboxOnce(repo, handler, cfg.BatchSize)
		}
	}()
	logger.Info("outbox job started",
		zap.Duration("poll_interval", cfg.PollInterval),
		zap.Int("batch_size", cfg.BatchSize))
}

func RunOutboxOnce(repo repository.OutboxEventRepository, handler OutboxHandler, batchSize int) {
	if handler == nil {
		handler = LogOutboxHandler{}
	}
	pending, err := repo.FindPending(batchSize)
	if err != nil {
		logger.Error("outbox find pending failed", err)
		return
	}
	for _, event := range pending {
		if claimErr := repo.UpdateStatus(event.Id, domain.OutboxStatusPending, domain.OutboxStatusProcessing); claimErr != nil {
			logger.Error("outbox claim failed", claimErr, zap.Int64("id", event.Id))
			continue
		}
		if handleErr := handler.Handle(event); handleErr != nil {
			logger.Error("outbox handler failed", handleErr, zap.Int64("id", event.Id))
			_ = repo.UpdateStatus(event.Id, domain.OutboxStatusProcessing, domain.OutboxStatusFailed)
			continue
		}
		if sentErr := repo.UpdateStatus(event.Id, domain.OutboxStatusProcessing, domain.OutboxStatusSent); sentErr != nil {
			logger.Error("outbox mark sent failed", sentErr, zap.Int64("id", event.Id))
		}
	}
}
