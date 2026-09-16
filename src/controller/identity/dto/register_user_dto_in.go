package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type RegisterUserDtoIn struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterUserDtoIn) ToDomain() *userdomain.UserDomain {
	return &userdomain.UserDomain{
		Name:     r.Name,
		Email:    r.Email,
		Password: r.Password,
	}
}

func (r *RegisterUserDtoIn) FromDomainUser(user *userdomain.UserDomain) *RegisterUserDtoIn {
	return &RegisterUserDtoIn{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}
}
