package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type UserProfileDtoOut struct {
	Id               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Email            string              `json:"email,omitempty"`
	ConfigVisibility ConfigVisibilityDto `json:"configVisibility"`
}

func (u UserProfileDtoOut) FromDomain(user *userdomain.UserDomain) UserProfileDtoOut {
	configVisibility := ConfigVisibilityDtoFromDomain(user.ConfigVisibility)
	if configVisibility == nil {
		configVisibility = ConfigVisibilityDto{}
	}
	return UserProfileDtoOut{
		Id:               user.Id,
		Name:             user.Name,
		Description:      user.Description,
		Email:            user.Email,
		ConfigVisibility: configVisibility,
	}
}
