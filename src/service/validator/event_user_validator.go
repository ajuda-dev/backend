package validator

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
)

type EventUserValidator interface {
	ValidateJoin(eventUser domain.EventUserDomain, category string) *rest_err.RestErr
	ValidateAddParticipant(eventUser domain.EventUserDomain, category string) *rest_err.RestErr
	ValidateUpdateParticipantStatus(status string) *rest_err.RestErr
}

type eventUserValidator struct{}

func NewEventUserValidator() EventUserValidator {
	return &eventUserValidator{}
}

func (e *eventUserValidator) ValidateJoin(eventUser domain.EventUserDomain, category string) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if eventUser.EventId == "" || !uuidv7.IsValidString(eventUser.EventId) {
		causes = append(causes, rest_err.Causes{
			Field:   "event_id",
			Message: "EventId is not valid",
		})
	}
	if eventUser.UserId == "" || !uuidv7.IsValidString(eventUser.UserId) {
		causes = append(causes, rest_err.Causes{
			Field:   "user_id",
			Message: "UserId is not valid",
		})
	}
	if category == domain.CategoryMentoring {
		causes = append(causes, rest_err.Causes{
			Field:   "event_id",
			Message: "MENTORING events cannot be joined, the host must invite the mentee",
		})
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			causes,
		)
	}
	return nil
}

func (e *eventUserValidator) ValidateAddParticipant(eventUser domain.EventUserDomain, category string) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if eventUser.EventId == "" || !uuidv7.IsValidString(eventUser.EventId) {
		causes = append(causes, rest_err.Causes{
			Field:   "event_id",
			Message: "EventId is not valid",
		})
	}
	if eventUser.UserId == "" || !uuidv7.IsValidString(eventUser.UserId) {
		causes = append(causes, rest_err.Causes{
			Field:   "user_id",
			Message: "UserId is not valid",
		})
	}
	if eventUser.Role != domain.RoleMentee && eventUser.Role != domain.RoleSpeaker {
		causes = append(causes, rest_err.Causes{
			Field:   "role",
			Message: "Role is not valid, use MENTEE or SPEAKER",
		})
	}
	if eventUser.Role == domain.RoleMentee && category != domain.CategoryMentoring {
		causes = append(causes, rest_err.Causes{
			Field:   "role",
			Message: "MENTEE role is only allowed for MENTORING events",
		})
	}
	if eventUser.Role == domain.RoleSpeaker && category == domain.CategoryMentoring {
		causes = append(causes, rest_err.Causes{
			Field:   "role",
			Message: "SPEAKER role is not allowed for MENTORING events",
		})
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			causes,
		)
	}
	return nil
}

func (e *eventUserValidator) ValidateUpdateParticipantStatus(status string) *rest_err.RestErr {
	if status != domain.StatusConfirmed && status != domain.StatusRejected {
		return rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			[]rest_err.Causes{{
				Field:   "status",
				Message: "Status is not valid, use CONFIRMED or REJECTED",
			}},
		)
	}
	return nil
}
