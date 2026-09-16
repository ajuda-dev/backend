package validator

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/ajuda-dev/backend/src/service/validation"
)

type UserValidator interface {
	ValidateRegisterUser(registerUser userdomain.UserDomain) *rest_err.RestErr
	ValidateUpdateUser(user userdomain.UserDomain) *rest_err.RestErr
}

type userValidator struct{}

func NewUserValidator() UserValidator {
	return &userValidator{}
}

func (u *userValidator) ValidateRegisterUser(registerUser userdomain.UserDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if !validation.IsValidName(registerUser.Name, false) {
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
	if !validation.IsValidEmail(registerUser.Email) {
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

func (u *userValidator) ValidateUpdateUser(user userdomain.UserDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	name := strings.TrimSpace(user.Name)
	description := strings.TrimSpace(user.Description)
	if user.Email != "" || user.Password != "" {
		causes = append(causes, rest_err.Causes{
			Field:   "body",
			Message: "email and password cannot be changed by this endpoint",
		})
	} else if name == "" && description == "" && len(user.ConfigVisibility) == 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "body",
			Message: "provide at least one field to update",
		})
	} else if name != "" && !validation.IsValidName(user.Name, false) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}

	if description != "" && len(description) > 500 {
		causes = append(causes, rest_err.Causes{
			Field:   "description",
			Message: "description must have at most 500 characters",
		})
	}
	causes = append(causes, validateConfigVisibility(user.ConfigVisibility)...)

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid user data", causes)
	}
	return nil
}

func validateConfigVisibility(config userdomain.ConfigVisibility) []rest_err.Causes {
	causes := []rest_err.Causes{}
	for key, item := range config {
		field := "config_visibility." + key
		value := strings.TrimSpace(item.Value)
		switch key {
		case userdomain.VisibilityKeyEmail:
			if item.Value != "" {
				causes = append(causes, rest_err.Causes{
					Field:   field,
					Message: "email value is managed by the system",
				})
			}
		case userdomain.VisibilityKeyGithub, userdomain.VisibilityKeyLinkedin, userdomain.VisibilityKeyOtherlink, userdomain.VisibilityKeyPhoto:
			if value != "" {
				if !validation.IsValidHTTPURL(value) {
					causes = append(causes, rest_err.Causes{
						Field:   field + ".value",
						Message: "value must be a valid http or https url",
					})
				} else if len(value) > 500 {
					causes = append(causes, rest_err.Causes{
						Field:   field + ".value",
						Message: "value must have at most 500 characters",
					})
				}
			} else if item.ShareWithCommunity {
				causes = append(causes, rest_err.Causes{
					Field:   field,
					Message: "shareWithCommunity requires a value",
				})
			}
		case userdomain.VisibilityKeyPhone:
			if value != "" {
				if !validation.IsValidPhone(value) {
					causes = append(causes, rest_err.Causes{
						Field:   field + ".value",
						Message: "value must be a valid phone number",
					})
				} else if len(value) > 20 {
					causes = append(causes, rest_err.Causes{
						Field:   field + ".value",
						Message: "value must have at most 20 characters",
					})
				}
			} else if item.ShareWithCommunity {
				causes = append(causes, rest_err.Causes{
					Field:   field,
					Message: "shareWithCommunity requires a value",
				})
			}
		default:
			causes = append(causes, rest_err.Causes{
				Field:   field,
				Message: "unsupported visibility key",
			})
		}
	}
	return causes
}
