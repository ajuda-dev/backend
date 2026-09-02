package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type LoginUserDtoOut struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (l *LoginUserDtoOut) FromDomainUser(user *domain.UserDomain, token string) *LoginUserDtoOut {
	return &LoginUserDtoOut{
		Token: token,
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}
}
