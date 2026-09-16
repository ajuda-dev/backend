package validator

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/samborkent/uuidv7"
)

type SkillUserValidator interface {
	ValidateAssign(skillUser skilldomain.SkillUserDomain) *rest_err.RestErr
}

type skillUserValidator struct{}

func NewSkillUserValidator() SkillUserValidator {
	return &skillUserValidator{}
}

func (s *skillUserValidator) ValidateAssign(skillUser skilldomain.SkillUserDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if skillUser.SkillId == "" || !uuidv7.IsValidString(skillUser.SkillId) {
		causes = append(causes, rest_err.Causes{
			Field:   "skill_id",
			Message: "SkillId is not valid",
		})
	}
	if skillUser.UserId == "" || !uuidv7.IsValidString(skillUser.UserId) {
		causes = append(causes, rest_err.Causes{
			Field:   "user_id",
			Message: "UserId is not valid",
		})
	}
	if skillUser.Level != skilldomain.LevelWantToLearn &&
		skillUser.Level != skilldomain.LevelLearnAndTeach &&
		skillUser.Level != skilldomain.LevelTeach {
		causes = append(causes, rest_err.Causes{
			Field:   "level",
			Message: "Level is not valid, use WANT_TO_LEARN, LEARN_AND_TEACH or TEACH",
		})
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid skill user data",
			causes,
		)
	}
	return nil
}
