package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type RegisterUserDtoOut struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r *RegisterUserDtoOut) FromDomainUser(user *domain.UserDomain) *RegisterUserDtoOut {
	return &RegisterUserDtoOut{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}
}