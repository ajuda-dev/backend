package job

import (
	"time"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PurgeConfig struct {
	Enabled       bool
	OlderThanDays int
	IntervalHours int
}

func StartPurgeJob(db *gorm.DB, cfg PurgeConfig) {
	if !cfg.Enabled {
		logger.Info("purge job disabled")
		return
	}
	ticker := time.NewTicker(time.Duration(cfg.IntervalHours) * time.Hour)
	go func() {
		for range ticker.C {
			RunPurge(db, cfg.OlderThanDays)
		}
	}()
	logger.Info("purge job started",
		zap.Int("older_than_days", cfg.OlderThanDays),
		zap.Int("interval_hours", cfg.IntervalHours))
}

// RunPurge é exportada para testes e para agendamento externo.
func RunPurge(db *gorm.DB, olderThanDays int) {
	if olderThanDays < 0 {
		olderThanDays = 0
	}
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	purgeEventUsers(db, cutoff)
	purgeEvents(db, cutoff)
	purgeSkills(db, cutoff)
	purgeCommunities(db, cutoff)
	purgeAddresses(db, cutoff)
	purgeUsers(db, cutoff)
}

func purgeEventUsers(db *gorm.DB, cutoff time.Time) {
	result := db.
		Where("status IN ? AND updated_at < ?", []string{domain.StatusCancelled, domain.StatusRejected}, cutoff).
		Delete(&entity.EventUserEntity{})
	logPurgeResult(result, "event_users")
}

func purgeEvents(db *gorm.DB, cutoff time.Time) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
		Delete(&entity.EventEntity{})
	logPurgeResult(result, "events")
}

func purgeSkills(db *gorm.DB, cutoff time.Time) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
		Delete(&entity.SkillEntity{})
	logPurgeResult(result, "skills")
}

func purgeCommunities(db *gorm.DB, cutoff time.Time) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM events e WHERE e.community_id = community.id)").
		Delete(&entity.CommunityEntity{})
	logPurgeResult(result, "community")
}

func purgeAddresses(db *gorm.DB, cutoff time.Time) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM community c WHERE c.address_id = addresses.id)").
		Where("NOT EXISTS (SELECT 1 FROM events e WHERE e.address_id = addresses.id)").
		Delete(&entity.AddressEntity{})
	logPurgeResult(result, "addresses")
}

func purgeUsers(db *gorm.DB, cutoff time.Time) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM community c WHERE c.owner_id = users.id)").
		Where("NOT EXISTS (SELECT 1 FROM events e WHERE e.owner_id = users.id)").
		Delete(&entity.UserEntity{})
	logPurgeResult(result, "users")
}

func logPurgeResult(result *gorm.DB, table string) {
	if result.Error != nil {
		logger.Error("purge error on "+table, result.Error)
		return
	}
	logger.Info("purge finished", zap.String("table", table), zap.Int64("rows", result.RowsAffected))
}
