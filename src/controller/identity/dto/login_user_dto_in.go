package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type LoginUserDtoIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginUserDtoIn) ToDomain() *domain.UserDomain {
	return &domain.UserDomain{
		Email:    l.Email,
		Password: l.Password,
	}
}
