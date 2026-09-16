package dto

import (
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
)

type RegisterSkillDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (r *RegisterSkillDto) ToDomain() *skilldomain.SkillDomain {
	return &skilldomain.SkillDomain{
		Name: r.Name,
	}
}

func (r RegisterSkillDto) FromDomain(skill *skilldomain.SkillDomain) RegisterSkillDto {
	return RegisterSkillDto{
		Id:   skill.Id,
		Name: skill.Name,
	}
}

type UpdateSkillDto struct {
	Name string `json:"name"`
}

func (u *UpdateSkillDto) ToDomain() *skilldomain.SkillDomain {
	return &skilldomain.SkillDomain{
		Name: u.Name,
	}
}
