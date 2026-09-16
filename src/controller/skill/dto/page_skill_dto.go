package dto

import (
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
)

type PageableSkillDto struct {
	HasNext bool       `json:"has_next"`
	Data    []SkillDto `json:"data"`
}

type SkillDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (s SkillDto) FromDomain(skill *skilldomain.SkillDomain) SkillDto {
	return SkillDto{
		Id:   skill.Id,
		Name: skill.Name,
	}
}

func (p PageableSkillDto) FromDomain(page skilldomain.PageableSkill) *PageableSkillDto {
	skills := make([]SkillDto, len(page.Data))
	for i, s := range page.Data {
		skills[i] = SkillDto{}.FromDomain(s)
	}
	return &PageableSkillDto{
		HasNext: page.HasNext,
		Data:    skills,
	}
}
