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
	err = db.AutoMigrate(&entity.UserEntity{}, &entity.AddressEntity{}, &entity.CommunityEntity{}, &entity.EventEntity{}, &entity.EventUserEntity{})
	if err != nil {
		return nil, fmt.Errorf("error doing AutoMigrate: %w", err)
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
