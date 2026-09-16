package dto

import (
	skilldto "github.com/ajuda-dev/backend/src/controller/skill/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type PageableUserDto struct {
	HasNext bool           `json:"has_next"`
	Data    []UserSkillDto `json:"data"`
}

type UserSkillDto struct {
	Id     string              `json:"id"`
	Name   string              `json:"name"`
	Skills []skilldto.SkillDto `json:"skills"`
}

func (u UserSkillDto) FromDomain(user *userdomain.UserDomain) UserSkillDto {
	skills := make([]skilldto.SkillDto, len(user.Skills))
	for i, s := range user.Skills {
		skills[i] = skilldto.SkillDto{}.FromDomain(&s)
	}
	return UserSkillDto{
		Id:     user.Id,
		Name:   user.Name,
		Skills: skills,
	}
}

func (p PageableUserDto) FromDomain(page userdomain.PageableUser) *PageableUserDto {
	users := make([]UserSkillDto, len(page.Data))
	for i, u := range page.Data {
		users[i] = UserSkillDto{}.FromDomain(u)
	}
	return &PageableUserDto{
		HasNext: page.HasNext,
		Data:    users,
	}
}
