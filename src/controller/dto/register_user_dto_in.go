package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type RegisterUserDtoIn struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterUserDtoIn) ToDomain() *domain.UserDomain {
	return &domain.UserDomain{
		Name:     r.Name,
		Email:    r.Email,
		Password: r.Password,
	}
}

func (r *RegisterUserDtoIn) FromDomainUser(user *domain.UserDomain) *RegisterUserDtoIn {
	return &RegisterUserDtoIn{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}
}
