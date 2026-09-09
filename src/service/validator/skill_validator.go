package validator

import (
	"regexp"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/samborkent/uuidv7"
)

var skillNameRegex = regexp.MustCompile(`^[A-ZÀ-Ü0-9][A-ZÀ-Ü0-9 .#+_-]*$`)

type SkillValidator interface {
	ValidateSkillName(name string) *rest_err.RestErr
	ValidateSkillId(id string) *rest_err.RestErr
}

type skillValidator struct{}

func NewSkillValidator() SkillValidator {
	return &skillValidator{}
}

func (s *skillValidator) ValidateSkillName(name string) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if name == "" || len(name) > 50 || !skillNameRegex.MatchString(name) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid skill data",
			causes,
		)
	}
	return nil
}

func (s *skillValidator) ValidateSkillId(id string) *rest_err.RestErr {
	if id == "" || !uuidv7.IsValidString(id) {
		return rest_err.NewBadRequestValidationError(
			"Invalid skill data",
			[]rest_err.Causes{{
				Field:   "id",
				Message: "Id is not valid",
			}},
		)
	}
	return nil
}
