package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/repository/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
	"github.com/samborkent/uuidv7"
)

type UserRepository interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
}

type userRepository struct {
	database *gorm.DB
}

// CreateUser implements UserRepository.
func (u *userRepository) CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity = entity.FromDomainUser(user)
	userEntity.Id = uuidv7.New().String()

	if err := u.database.Create(&userEntity).Error; err != nil {
		return &domain.UserDomain{}, rest_err.NewInternalServerError(err.Error())
	}

	return userEntity.ToDomainUser(), nil
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		database: db,
	}
}
