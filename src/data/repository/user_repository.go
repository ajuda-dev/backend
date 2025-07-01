package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
	GetUserByEmail(email string) (*domain.UserDomain, *rest_err.RestErr)
	FindById(id string) (*domain.UserDomain, *rest_err.RestErr)
}

type userRepository struct {
	database *gorm.DB
}

// FindById implements UserRepository.
func (u *userRepository) FindById(id string) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity entity.UserEntity
	if err := u.database.Where("id = ?", id).First(&userEntity).Error ; err != nil {
		handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		database: db,
	}
}

func (u *userRepository) CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity = entity.FromDomainUser(user)
	userEntity.Id = uuidv7.New().String()

	if err := u.database.Create(&userEntity).Error; err != nil {
		return &domain.UserDomain{}, rest_err.NewInternalServerError(err.Error())
	}

	return userEntity.ToDomainUser(), nil
}
func (u *userRepository) GetUserByEmail(email string) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity entity.UserEntity
	if err := u.database.Where("email = ?", email).First(&userEntity).Error; err != nil {
		return handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func handlerErrorDataBase(err error) (*domain.UserDomain, *rest_err.RestErr) {
	if err == gorm.ErrRecordNotFound {
		return nil, rest_err.NewNotFoundError("User not found")
	}
	return nil, rest_err.NewInternalServerError(err.Error())
}
