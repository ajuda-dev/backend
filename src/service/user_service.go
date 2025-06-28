package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
	"golang.org/x/crypto/bcrypt"
)

func NewUserService(userRepository repository.UserRepository, validator validator.UserValidator) UserService {
	return &userService{
		userRepository: userRepository,
		validator:      validator,
	}
}

type UserService interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
}

type userService struct {
	userRepository repository.UserRepository
	validator      validator.UserValidator
}

// CreateUser implements UserService.
func (u *userService) CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	err := u.validator.ValidateRegisterUser(*user)
	if err != nil {
		return nil, err
	}
	existingUser , err := u.userRepository.GetUserByEmail(user.Email)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, rest_err.NewInternalServerError(err.Error())
	}	

	if existingUser != nil {
		return nil, rest_err.NewBadRequestValidationError(	
			"Invalid user data",
			[]rest_err.Causes{ 
				{
					Field:   "email",	
					Message: "Email already exists",
				},
			})
	}
	password, errH := hashPassword(user.Password)
	if errH != nil {
		return nil, rest_err.NewInternalServerError(errH.Error())
	}
	user.Password = password
	user, err = u.userRepository.CreateUser(user)
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
