package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"golang.org/x/crypto/bcrypt"
)

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

type UserService interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
}

type userService struct {
	userRepository repository.UserRepository
}

// CreateUser implements UserService.
func (u *userService) CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	password, errH := hashPassword(user.Password)
	if errH != nil {
		return nil, rest_err.NewInternalServerError( errH.Error())
	}
	user.Password = password
	user, err := u.userRepository.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}


func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}


// func checkPasswordHash(password, hash string) bool {
// 	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
// 	return err == nil
// }
