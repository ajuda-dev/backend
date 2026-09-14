package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type LoginUserDtoOut struct {
	Token string `json:"token,omitempty"`
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (l *LoginUserDtoOut) FromDomainUser(user *domain.UserDomain) *LoginUserDtoOut {
	return &LoginUserDtoOut{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}
}
