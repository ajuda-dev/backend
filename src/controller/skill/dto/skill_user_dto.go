package dto

import (
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
)

type SkillUserDto struct {
	Id      string    `json:"id"`
	SkillId string    `json:"skill_id"`
	UserId  string    `json:"user_id"`
	Level   string    `json:"level"`
	Skill   *SkillDto `json:"skill,omitempty"`
}

func (s SkillUserDto) FromDomain(skillUser *skilldomain.SkillUserDomain) SkillUserDto {
	dtoSkillUser := SkillUserDto{
		Id:      skillUser.Id,
		SkillId: skillUser.SkillId,
		UserId:  skillUser.UserId,
		Level:   skillUser.Level,
	}
	if skillUser.Skill != nil {
		skill := SkillDto{}.FromDomain(skillUser.Skill)
		dtoSkillUser.Skill = &skill
	}
	return dtoSkillUser
}

func ToSkillUserDtoList(skillUsers []*skilldomain.SkillUserDomain) []SkillUserDto {
	dtos := make([]SkillUserDto, len(skillUsers))
	for i, su := range skillUsers {
		dtos[i] = SkillUserDto{}.FromDomain(su)
	}
	return dtos
}

type AssignSkillDto struct {
	UserId string `json:"user_id"`
	Level  string `json:"level"`
}

func (a *AssignSkillDto) ToDomain(skillId string) *skilldomain.SkillUserDomain {
	return &skilldomain.SkillUserDomain{
		SkillId: skillId,
		UserId:  a.UserId,
		Level:   a.Level,
	}
}
