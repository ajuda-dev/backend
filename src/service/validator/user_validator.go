package validator

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type UserValidator interface {
	ValidateRegisterUser(registerUser domain.UserDomain) *rest_err.RestErr
	ValidateUpdateUser(user domain.UserDomain) *rest_err.RestErr
}

type userValidator struct{}



func NewUserValidator() UserValidator {
	return &userValidator{}
}


func (u *userValidator) ValidateRegisterUser(registerUser domain.UserDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if !isValidName(registerUser.Name, false) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}
	if len(registerUser.Password) < 6 {
		causes = append(causes, rest_err.Causes{
			Field:   "password",
			Message: "Password must be at least 6 characters long",
		})
	}
	if registerUser.Password == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "password",
			Message: "Password cannot be empty",
		})
	}
	if registerUser.Email == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "email",
			Message: "Email cannot be empty",
		})
	}
	if !isValidEmail(registerUser.Email) {
		causes = append(causes, rest_err.Causes{
			Field:   "email",
			Message: "Email is not valid",
		})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid user data",
			causes,
		)
	}
	return nil
}

func (u *userValidator) ValidateUpdateUser(user domain.UserDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	name := strings.TrimSpace(user.Name)
	if user.Email != "" || user.Password != "" {
		causes = append(causes, rest_err.Causes{
			Field:   "body",
			Message: "email and password cannot be changed by this endpoint",
		})
	} else if name == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "body",
			Message: "provide at least one field to update",
		})
	} else if !isValidName(user.Name, false) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid user data", causes)
	}
	return nil
}



