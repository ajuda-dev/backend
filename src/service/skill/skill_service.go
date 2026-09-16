package skill

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	skillrepo "github.com/ajuda-dev/backend/src/data/skill/repository"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	skillvalidator "github.com/ajuda-dev/backend/src/service/skill/validator"
)

type SkillService interface {
	CreateSkill(skill *skilldomain.SkillDomain) (*skilldomain.SkillDomain, *rest_err.RestErr)
	GetSkillById(id string) (*skilldomain.SkillDomain, *rest_err.RestErr)
	GetAll(filter skillrepo.SkillFilter, page int, limit int) (*skilldomain.PageableSkill, *rest_err.RestErr)
	UpdateSkill(id string, name string, requesterId string) (*skilldomain.SkillDomain, *rest_err.RestErr)
	DeleteSkill(id string, requesterId string) *rest_err.RestErr
}

type skillService struct {
	userService     identity.UserService
	skillRepository skillrepo.SkillRepository
	skillValidator  skillvalidator.SkillValidator
}

func NewSkillService(userService identity.UserService, skillRepository skillrepo.SkillRepository, skillValidator skillvalidator.SkillValidator) SkillService {
	return &skillService{
		userService:     userService,
		skillRepository: skillRepository,
		skillValidator:  skillValidator,
	}
}

func normalizeSkillName(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func (s *skillService) CreateSkill(skill *skilldomain.SkillDomain) (*skilldomain.SkillDomain, *rest_err.RestErr) {
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

func (s *skillService) GetSkillById(id string) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	return s.skillRepository.FindById(id)
}

func (s *skillService) GetAll(filter skillrepo.SkillFilter, page int, limit int) (*skilldomain.PageableSkill, *rest_err.RestErr) {
	filter.Name = normalizeSkillName(filter.Name)
	return s.skillRepository.FindAll(filter, page, limit)
}

func (s *skillService) UpdateSkill(id string, name string, requesterId string) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	if err := s.skillValidator.ValidateSkillId(id); err != nil {
		return nil, err
	}
	requester, err := identity.AuthenticatedUser(s.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if !identity.RoleAtLeast(requester.Role, userdomain.UserRoleModerator) {
		return nil, rest_err.NewForbiddenError("only moderators and admins can update skills")
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

func (s *skillService) DeleteSkill(id string, requesterId string) *rest_err.RestErr {
	if err := s.skillValidator.ValidateSkillId(id); err != nil {
		return err
	}
	requester, err := identity.AuthenticatedUser(s.userService, requesterId)
	if err != nil {
		return err
	}
	if !identity.RoleAtLeast(requester.Role, userdomain.UserRoleModerator) {
		return rest_err.NewForbiddenError("only moderators and admins can delete skills")
	}
	return s.skillRepository.SoftDeleteById(id)
}
