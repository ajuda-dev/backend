package dto

import (
	"github.com/ajuda-dev/backend/src/service/domain"
)

type PageableUserDto struct {
	HasNext bool           `json:"has_next"`
	Data    []UserSkillDto `json:"data"`
}

type UserSkillDto struct {
	Id     string     `json:"id"`
	Name   string     `json:"name"`
	Email  string     `json:"email"`
	Skills []SkillDto `json:"skills"`
}

func (u UserSkillDto) FromDomain(user *domain.UserDomain) UserSkillDto {
	skills := make([]SkillDto, len(user.Skills))
	for i, s := range user.Skills {
		skills[i] = SkillDto{}.FromDomain(&s)
	}
	return UserSkillDto{
		Id:     user.Id,
		Name:   user.Name,
		Email:  user.Email,
		Skills: skills,
	}
}

func (p PageableUserDto) FromDomain(page domain.PageableUser) *PageableUserDto {
	users := make([]UserSkillDto, len(page.Data))
	for i, u := range page.Data {
		users[i] = UserSkillDto{}.FromDomain(u)
	}
	return &PageableUserDto{
		HasNext: page.HasNext,
		Data:    users,
	}
}
