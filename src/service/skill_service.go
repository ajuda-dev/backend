package service

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type SkillService interface {
	CreateSkill(skill *domain.SkillDomain) (*domain.SkillDomain, *rest_err.RestErr)
	GetSkillById(id string) (*domain.SkillDomain, *rest_err.RestErr)
	GetAll(filter repository.SkillFilter, page int, limit int) (*domain.PageableSkill, *rest_err.RestErr)
	UpdateSkill(id string, name string) (*domain.SkillDomain, *rest_err.RestErr)
	DeleteSkill(id string) *rest_err.RestErr
}

type skillService struct {
	skillRepository repository.SkillRepository
	skillValidator  validator.SkillValidator
}

func NewSkillService(skillRepository repository.SkillRepository, skillValidator validator.SkillValidator) SkillService {
	return &skillService{
		skillRepository: skillRepository,
		skillValidator:  skillValidator,
	}
}

func normalizeSkillName(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func (s *skillService) CreateSkill(skill *domain.SkillDomain) (*domain.SkillDomain, *rest_err.RestErr) {
	skill.Name = normalizeSkillName(skill.Name)
	if err := s.skillValidator.ValidateSkillName(skill.Name); err != nil {
		return nil, err
	}
	existing, err := s.skillRepository.FindByName(skill.Name)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}
	if existing != nil {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid skill data",
			[]rest_err.Causes{{
				Field:   "name",
				Message: "Skill already exists",
			}})
	}
	return s.skillRepository.CreateSkill(skill)
}

func (s *skillService) GetSkillById(id string) (*domain.SkillDomain, *rest_err.RestErr) {
	return s.skillRepository.FindById(id)
}

func (s *skillService) GetAll(filter repository.SkillFilter, page int, limit int) (*domain.PageableSkill, *rest_err.RestErr) {
	filter.Name = normalizeSkillName(filter.Name)
	return s.skillRepository.FindAll(filter, page, limit)
}

func (s *skillService) UpdateSkill(id string, name string) (*domain.SkillDomain, *rest_err.RestErr) {
	if err := s.skillValidator.ValidateSkillId(id); err != nil {
		return nil, err
	}
	name = normalizeSkillName(name)
	if err := s.skillValidator.ValidateSkillName(name); err != nil {
		return nil, err
	}
	if _, err := s.skillRepository.FindById(id); err != nil {
		return nil, err
	}
	existing, err := s.skillRepository.FindByName(name)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}
	if existing != nil && existing.Id != id {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid skill data",
			[]rest_err.Causes{{
				Field:   "name",
				Message: "Skill already exists",
			}})
	}
	return s.skillRepository.UpdateName(id, name)
}

func (s *skillService) DeleteSkill(id string) *rest_err.RestErr {
	if err := s.skillValidator.ValidateSkillId(id); err != nil {
		return err
	}
	return s.skillRepository.SoftDeleteById(id)
}
