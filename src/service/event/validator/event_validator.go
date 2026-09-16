package validator

import (
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
)

type EventValidator interface {
	ValidatorRegisterEvent(event domain.EventDomain) *rest_err.RestErr
}

type eventValidator struct{}

func NewEventValidator() EventValidator {
	return &eventValidator{}
}

func (e *eventValidator) ValidatorRegisterEvent(event domain.EventDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if !isValidName(event.Title, true) {
		causes = append(causes, rest_err.Causes{
			Field:   "title",
			Message: "Title is not valid",
		})
	}
	if event.Category != domain.CategoryCommunityEvent &&
		event.Category != domain.CategoryMentoring &&
		event.Category != domain.CategoryWebinar {
		causes = append(causes, rest_err.Causes{
			Field:   "category",
			Message: "Category is not valid",
		})
	}
	if event.Type != domain.TypeOnline &&
		event.Type != domain.TypeInperson &&
		event.Type != domain.TypeHybrid {
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
	if event.Type == domain.TypeInperson || event.Type == domain.TypeHybrid {
		if event.Address == nil {
			causes = append(causes, rest_err.Causes{
				Field:   "address_id",
				Message: "AddressId is required for " + event.Type + " events",
			})
		}
	}
	if event.Type == domain.TypeOnline && event.Address != nil {
		causes = append(causes, rest_err.Causes{
			Field:   "address_id",
			Message: "AddressId must be empty for ONLINE events",
		})
	}
	if event.CreatorRole != "" {
		if event.Category != domain.CategoryMentoring {
			causes = append(causes, rest_err.Causes{
				Field:   "creator_role",
				Message: "CreatorRole is only allowed for MENTORING events",
			})
		} else if event.CreatorRole != domain.RoleMentor && event.CreatorRole != domain.RoleMentee {
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
