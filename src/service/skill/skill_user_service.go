package skill

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	skillrepo "github.com/ajuda-dev/backend/src/data/skill/repository"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	skillvalidator "github.com/ajuda-dev/backend/src/service/skill/validator"
)

type SkillUserService interface {
	AssignSkill(skillUser *skilldomain.SkillUserDomain, requesterId string) (*skilldomain.SkillUserDomain, *rest_err.RestErr)
	GetUserSkills(userId string) ([]*skilldomain.SkillUserDomain, *rest_err.RestErr)
	RemoveSkillFromUser(userId string, skillId string, requesterId string) *rest_err.RestErr
}

type skillUserService struct {
	userService         identity.UserService
	skillService        SkillService
	skillUserRepository skillrepo.SkillUserRepository
	skillUserValidator  skillvalidator.SkillUserValidator
}

func NewSkillUserService(
	userService identity.UserService,
	skillService SkillService,
	skillUserRepository skillrepo.SkillUserRepository,
	skillUserValidator skillvalidator.SkillUserValidator) SkillUserService {
	return &skillUserService{
		userService:         userService,
		skillService:        skillService,
		skillUserRepository: skillUserRepository,
		skillUserValidator:  skillUserValidator,
	}
}

func (s *skillUserService) requireSelfOrAdmin(requesterId, targetId string) *rest_err.RestErr {
	requester, err := identity.AuthenticatedUser(s.userService, requesterId)
	if err != nil {
		return err
	}
	if requester.Id != targetId && requester.Role != userdomain.UserRoleAdmin {
		return rest_err.NewForbiddenError("only the user themselves or an admin can manage this user's skills")
	}
	return nil
}

func (s *skillUserService) AssignSkill(skillUser *skilldomain.SkillUserDomain, requesterId string) (*skilldomain.SkillUserDomain, *rest_err.RestErr) {
	if err := s.skillUserValidator.ValidateAssign(*skillUser); err != nil {
		return nil, err
	}
	if err := s.requireSelfOrAdmin(requesterId, skillUser.UserId); err != nil {
		return nil, err
	}
	if _, err := s.skillService.GetSkillById(skillUser.SkillId); err != nil {
		if err.Code == rest_err.NOT_FOUND {
			return nil, rest_err.NewBadRequestValidationError(
				"Invalid skill user data",
				[]rest_err.Causes{{
					Field:   "skill_id",
					Message: "skill_id is not valid, not found this skill",
				}})
		}
		return nil, err
	}
	if _, err := s.userService.FindById(skillUser.UserId); err != nil {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid skill user data",
			[]rest_err.Causes{{
				Field:   "user_id",
				Message: "user_id is not valid, not found this user",
			}})
	}
	return s.skillUserRepository.Create(skillUser)
}

func (s *skillUserService) GetUserSkills(userId string) ([]*skilldomain.SkillUserDomain, *rest_err.RestErr) {
	return s.skillUserRepository.FindByUser(userId)
}

func (s *skillUserService) RemoveSkillFromUser(userId string, skillId string, requesterId string) *rest_err.RestErr {
	if err := s.requireSelfOrAdmin(requesterId, userId); err != nil {
		return err
	}
	return s.skillUserRepository.DeleteByUserAndSkill(userId, skillId)
}
