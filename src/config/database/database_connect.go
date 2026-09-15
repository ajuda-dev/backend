package database

import (
	"fmt"
	"os"
	"strings"

	"github.com/ajuda-dev/backend/src/data/entity"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB_HOST      = "DB_HOST"
	DB_USER      = "DB_USER"
	DB_PASSWORD  = "DB_PASSWORD"
	DB_NAME      = "DB_NAME"
	DB_SSL_MODE  = "DB_SSL_MODE"
	DB_TIME_ZONE = "DB_TIME_ZONE"
)

func Connect() (db *gorm.DB, err error) {
	host, port := splitHostPort(os.Getenv(DB_HOST))

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host,
		port,
		os.Getenv(DB_USER),
		os.Getenv(DB_PASSWORD),
		os.Getenv(DB_NAME),
		os.Getenv(DB_SSL_MODE),
		os.Getenv(DB_TIME_ZONE),
	)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// unaccent é usado pela busca por cidade e nome (planos 13/15/16); é uma extensão
	// "trusted", então o dono do banco a cria sem superusuário. Falha aqui é fail-fast:
	// sem a extensão, toda busca por cidade/nome responderia 500.
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS unaccent").Error; err != nil {
		return nil, fmt.Errorf("error enabling unaccent extension: %w", err)
	}
	err = db.AutoMigrate(&entity.UserEntity{}, &entity.AddressEntity{}, &entity.CommunityEntity{}, &entity.EventEntity{}, &entity.EventUserEntity{}, &entity.SkillEntity{}, &entity.SkillUserEntity{}, &entity.CommunityUserEntity{}, &entity.OAuthAccountEntity{}, &entity.OutboxEventEntity{}, &entity.EmailCodeEntity{})
	if err != nil {
		return nil, fmt.Errorf("error doing AutoMigrate: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_outbox_events_user_read_created ON outbox_events (user_id, read_at, created_at)").Error; err != nil {
		return nil, fmt.Errorf("error creating outbox inbox index: %w", err)
	}

	return db, nil
}

func splitHostPort(hostPort string) (host, port string) {
	host = hostPort
	port = "5432"
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		host = hostPort[:idx]
		port = hostPort[idx+1:]
	}
	return host, port
}
