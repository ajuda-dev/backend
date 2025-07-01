package database

import (
	"fmt"
	"os"

	"github.com/ajuda-dev/backend/src/data/entity"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	HOST= "HOST"
	USER= "USER"
	PASSWORD= "PASSWORD"
	DB_NAME= "DB_NAME"
	DB_PORT= "DB_PORT"
	DB_SSL_MODE= "DB_SSL_MODE"
	DB_TIME_ZONE= "DB_TIME_ZONE"
)



func Connect() (db *gorm.DB, err error) {
	
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		os.Getenv(HOST),
		os.Getenv(USER),
		os.Getenv(PASSWORD),
		os.Getenv(DB_NAME),
		os.Getenv(DB_PORT),
		os.Getenv(DB_SSL_MODE),
		os.Getenv(DB_TIME_ZONE),
	)
	

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&entity.UserEntity{}, &entity.AddressEntity{}, &entity.CommunityEntity{})
	if err != nil {
		return nil, fmt.Errorf("error doing AutoMigrate: %w", err)
	}

	return db, nil
}


