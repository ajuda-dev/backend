package dto

import (
	"github.com/ajuda-dev/backend/src/service/domain"
)

type RegisterSkillDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (r *RegisterSkillDto) ToDomain() *domain.SkillDomain {
	return &domain.SkillDomain{
		Name: r.Name,
	}
}

func (r RegisterSkillDto) FromDomain(skill *domain.SkillDomain) RegisterSkillDto {
	return RegisterSkillDto{
		Id:   skill.Id,
		Name: skill.Name,
	}
}

type UpdateSkillDto struct {
	Name string `json:"name"`
}

func (u *UpdateSkillDto) ToDomain() *domain.SkillDomain {
	return &domain.SkillDomain{
		Name: u.Name,
	}
}
