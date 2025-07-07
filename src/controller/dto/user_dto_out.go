package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type UserDtoOut struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r *UserDtoOut) FromDomainUser(user *domain.UserDomain) *UserDtoOut {
	return &UserDtoOut{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}
}
