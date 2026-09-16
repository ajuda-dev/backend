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
	} else if name != "" && !isValidName(user.Name, false) {
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

func validateConfigVisibility(config domain.ConfigVisibility) []rest_err.Causes {
	causes := []rest_err.Causes{}
	for key, item := range config {
		field := "config_visibility." + key
		value := strings.TrimSpace(item.Value)
		switch key {
		case domain.VisibilityKeyEmail:
			if item.Value != "" {
				causes = append(causes, rest_err.Causes{
					Field:   field,
					Message: "email value is managed by the system",
				})
			}
		case domain.VisibilityKeyGithub, domain.VisibilityKeyLinkedin, domain.VisibilityKeyOtherlink, domain.VisibilityKeyPhoto:
			if value != "" {
				if !isValidHTTPURL(value) {
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
		case domain.VisibilityKeyPhone:
			if value != "" {
				if !isValidPhone(value) {
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
