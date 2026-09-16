package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type UserDtoOut struct {
	Id               string              `json:"id"`
	Name             string              `json:"name"`
	Email            string              `json:"email"`
	Role             string              `json:"role"`
	EmailVerified    bool                `json:"emailVerified"`
	Token            string              `json:"token,omitempty"`
	Description      string              `json:"description,omitempty"`
	ConfigVisibility ConfigVisibilityDto `json:"configVisibility,omitempty"`
}

func (r *UserDtoOut) FromDomainUser(user *userdomain.UserDomain) *UserDtoOut {
	return &UserDtoOut{
		Id:            user.Id,
		Name:          user.Name,
		Email:         user.Email,
		Role:          user.Role,
		EmailVerified: user.EmailVerified(),
	}
}

func (r *UserDtoOut) FromDomainUserWithProfile(user *userdomain.UserDomain) *UserDtoOut {
	dtoOut := r.FromDomainUser(user)
	dtoOut.Description = user.Description
	dtoOut.ConfigVisibility = ConfigVisibilityDtoFromDomain(user.ConfigVisibility)
	return dtoOut
}
