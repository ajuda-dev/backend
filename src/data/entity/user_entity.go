package entity

import (
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)



// UserEntity represents the user entity in the database.
type UserEntity struct {
	gorm.Model
	Id       string `gorm:"primaryKey;type:uuid"`
	Name     string `gorm:"type:varchar(100);not null"`
	Email    string `gorm:"type:varchar(100);unique;not null"`
	Password string `gorm:"type:varchar(100);not null"`
}

// TableName overrides the table name used by UserEntity to `user`
func (UserEntity) TableName() string {
	return "users"
}




func (u *UserEntity) ToDomainUser() *domain.UserDomain {
	return &domain.UserDomain{
		Id:       u.Id,
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
	}
}

func FromDomainUser(user *domain.UserDomain) *UserEntity {
	return &UserEntity{
		Id:       user.Id,
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}
}