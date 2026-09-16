package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type LoginUserDtoOut struct {
	Token         string `json:"token,omitempty"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
}

func (l *LoginUserDtoOut) FromDomainUser(user *userdomain.UserDomain) *LoginUserDtoOut {
	return &LoginUserDtoOut{
		Id:            user.Id,
		Name:          user.Name,
		Email:         user.Email,
		Role:          user.Role,
		EmailVerified: user.EmailVerified(),
	}
}
