package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type LoginUserDtoIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginUserDtoIn) ToDomain() *userdomain.UserDomain {
	return &userdomain.UserDomain{
		Email:    l.Email,
		Password: l.Password,
	}
}
