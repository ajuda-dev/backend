package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type UserEntity struct {
	Id        string         `gorm:"primaryKey;type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:idx_users_email_del,priority:2"`
	Name      string         `gorm:"type:varchar(100);not null"`
	Email     string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_users_email_del,priority:1"`
	Password  string         `gorm:"type:varchar(100);not null"`
	Role      string         `gorm:"type:varchar(20);not null;default:('USER')"`
}


func (UserEntity) TableName() string {
	return "users"
}




func (u *UserEntity) ToDomainUser() *domain.UserDomain {
	return &domain.UserDomain{
		Id:       u.Id,
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
		Role:     u.Role,
	}
}

func FromDomainUser(user *domain.UserDomain) *UserEntity {
	return &UserEntity{
		Id:       user.Id,
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		Role:     user.Role,
	}
}