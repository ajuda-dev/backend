package validator

import (
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	"github.com/ajuda-dev/backend/src/service/validation"
	"github.com/samborkent/uuidv7"
)

type EventValidator interface {
	ValidatorRegisterEvent(event eventdomain.EventDomain) *rest_err.RestErr
	ValidateReschedule(startAt time.Time, comment string) *rest_err.RestErr
	ValidateCancelComment(comment string) *rest_err.RestErr
}

type eventValidator struct{}

func NewEventValidator() EventValidator {
	return &eventValidator{}
}

func (e *eventValidator) ValidatorRegisterEvent(event eventdomain.EventDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if !validation.IsValidName(event.Title, true) {
		causes = append(causes, rest_err.Causes{
			Field:   "title",
			Message: "Title is not valid",
		})
	}
	if event.Category != eventdomain.CategoryCommunityEvent &&
		event.Category != eventdomain.CategoryMentoring &&
		event.Category != eventdomain.CategoryWebinar {
		causes = append(causes, rest_err.Causes{
			Field:   "category",
			Message: "Category is not valid",
		})
	}
	if event.Type != eventdomain.TypeOnline &&
		event.Type != eventdomain.TypeInperson &&
		event.Type != eventdomain.TypeHybrid {
		causes = append(causes, rest_err.Causes{
			Field:   "type",
			Message: "Type is not valid",
		})
	}
	if event.Owner.Id == "" || !uuidv7.IsValidString(event.Owner.Id) {
		causes = append(causes, rest_err.Causes{
			Field:   "owner_id",
			Message: "OwnerId is not valid",
		})
	}
	if event.Community != nil && !uuidv7.IsValidString(event.Community.Id) {
		causes = append(causes, rest_err.Causes{
			Field:   "community_id",
			Message: "CommunityId is not valid",
		})
	}
	if event.Address != nil && !uuidv7.IsValidString(event.Address.Id) {
		causes = append(causes, rest_err.Causes{
			Field:   "address_id",
			Message: "AddressId is not valid",
		})
	}
	if event.StartAt.IsZero() || !event.StartAt.After(time.Now()) {
		causes = append(causes, rest_err.Causes{
			Field:   "start_at",
			Message: "StartAt must be in the future",
		})
	}
	if event.DurationMin <= 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "duration_min",
			Message: "DurationMin must be greater than zero",
		})
	}
	if event.Type == eventdomain.TypeInperson || event.Type == eventdomain.TypeHybrid {
		if event.Address == nil {
			causes = append(causes, rest_err.Causes{
				Field:   "address_id",
				Message: "AddressId is required for " + event.Type + " events",
			})
		}
	}
	if event.Type == eventdomain.TypeOnline && event.Address != nil {
		causes = append(causes, rest_err.Causes{
			Field:   "address_id",
			Message: "AddressId must be empty for ONLINE events",
		})
	}
	if event.CreatorRole != "" {
		if event.Category != eventdomain.CategoryMentoring {
			causes = append(causes, rest_err.Causes{
				Field:   "creator_role",
				Message: "CreatorRole is only allowed for MENTORING events",
			})
		} else if event.CreatorRole != eventdomain.RoleMentor && event.CreatorRole != eventdomain.RoleMentee {
			causes = append(causes, rest_err.Causes{
				Field:   "creator_role",
				Message: "CreatorRole is not valid, use MENTOR or MENTEE",
			})
		}
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid event data",
			causes,
		)
	}
	return nil
}

func (e *eventValidator) ValidateReschedule(startAt time.Time, comment string) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if startAt.IsZero() || !startAt.After(time.Now()) {
		causes = append(causes, rest_err.Causes{
			Field:   "start_at",
			Message: "StartAt must be in the future",
		})
	}
	causes = append(causes, commentCauses(comment, "comment is required when rescheduling")...)
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid event data", causes)
	}
	return nil
}

func (e *eventValidator) ValidateCancelComment(comment string) *rest_err.RestErr {
	causes := commentCauses(comment, "comment is required when cancelling")
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid event data", causes)
	}
	return nil
}

func commentCauses(comment string, requiredMessage string) []rest_err.Causes {
	if comment == "" {
		return []rest_err.Causes{{
			Field:   "comment",
			Message: requiredMessage,
		}}
	}
	if len(comment) > 500 {
		return []rest_err.Causes{{
			Field:   "comment",
			Message: "comment must have at most 500 characters",
		}}
	}
	return nil
}
