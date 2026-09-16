package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type UserEntity struct {
	Id               string `gorm:"primaryKey;type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt       `gorm:"uniqueIndex:idx_users_email_del,priority:2"`
	Name             string               `gorm:"type:varchar(100);not null"`
	Email            string               `gorm:"type:varchar(100);not null;uniqueIndex:idx_users_email_del,priority:1"`
	Password         string               `gorm:"type:varchar(100);not null"`
	Role             string               `gorm:"type:varchar(20);not null;default:('USER')"`
	Description      string               `gorm:"type:varchar(500)"`
	ConfigVisibility UserConfigVisibility `gorm:"type:jsonb;column:config_visibility"`
	EmailVerifiedAt  *time.Time           `gorm:"type:timestamptz"`
}

func (UserEntity) TableName() string {
	return "users"
}

func (u *UserEntity) ToDomainUser() *domain.UserDomain {
	config := u.ConfigVisibility.ToDomain()
	emailConfig := config[domain.VisibilityKeyEmail]
	emailConfig.Value = u.Email
	config[domain.VisibilityKeyEmail] = emailConfig

	return &domain.UserDomain{
		Id:               u.Id,
		Name:             u.Name,
		Email:            u.Email,
		Password:         u.Password,
		Role:             u.Role,
		Description:      u.Description,
		ConfigVisibility: config,
		EmailVerifiedAt:  u.EmailVerifiedAt,
	}
}

func FromDomainUser(user *domain.UserDomain) *UserEntity {
	return &UserEntity{
		Id:               user.Id,
		Name:             user.Name,
		Email:            user.Email,
		Password:         user.Password,
		Role:             user.Role,
		Description:      user.Description,
		ConfigVisibility: UserConfigVisibilityFromDomain(user.ConfigVisibility),
		EmailVerifiedAt:  user.EmailVerifiedAt,
	}
}
