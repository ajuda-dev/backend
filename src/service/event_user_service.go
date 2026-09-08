package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type EventUserService interface {
	JoinEvent(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr)
	AddParticipant(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr)
	GetParticipants(eventId string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr)
	UpdateParticipantStatus(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr)
	CancelParticipation(eventId string, userId string) (*domain.EventUserDomain, *rest_err.RestErr)
}

type eventUserService struct {
	userService         UserService
	eventService        EventService
	eventUserRepository repository.EventUserRepository
	eventUserValidator  validator.EventUserValidator
}

func NewEventUserService(
	userService UserService,
	eventService EventService,
	eventUserRepository repository.EventUserRepository,
	eventUserValidator validator.EventUserValidator) EventUserService {
	return &eventUserService{
		userService:         userService,
		eventService:        eventService,
		eventUserRepository: eventUserRepository,
		eventUserValidator:  eventUserValidator,
	}
}

func (e *eventUserService) JoinEvent(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr) {
	event, err := e.eventService.GetEventById(eventUser.EventId)
	if err != nil {
		return nil, err
	}
	eventUser.Role = domain.RoleAttendee
	eventUser.Status = domain.StatusConfirmed
	if err := e.eventUserValidator.ValidateJoin(*eventUser, event.Category); err != nil {
		return nil, err
	}
	if err := e.validateUserExists(eventUser.UserId); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) AddParticipant(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr) {
	event, err := e.eventService.GetEventById(eventUser.EventId)
	if err != nil {
		return nil, err
	}
	if err := e.eventUserValidator.ValidateAddParticipant(*eventUser, event.Category); err != nil {
		return nil, err
	}
	if eventUser.Role == domain.RoleMentee {
		eventUser.Status = domain.StatusRequested
	} else {
		eventUser.Status = domain.StatusConfirmed
	}
	if err := e.validateUserExists(eventUser.UserId); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) GetParticipants(eventId string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr) {
	if _, err := e.eventService.GetEventById(eventId); err != nil {
		return nil, err
	}
	return e.eventUserRepository.FindByEvent(eventId, status)
}

func (e *eventUserService) UpdateParticipantStatus(eventUser *domain.EventUserDomain) (*domain.EventUserDomain, *rest_err.RestErr) {
	event, err := e.eventService.GetEventById(eventUser.EventId)
	if err != nil {
		return nil, err
	}
	if err := e.eventUserValidator.ValidateUpdateParticipantStatus(eventUser.Status); err != nil {
		return nil, err
	}
	return e.eventUserRepository.UpdateStatus(eventUser.EventId, eventUser.UserId, eventUser.Status, event.MaxSlots)
}

func (e *eventUserService) CancelParticipation(eventId string, userId string) (*domain.EventUserDomain, *rest_err.RestErr) {
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	return e.eventUserRepository.UpdateStatus(eventId, userId, domain.StatusCancelled, event.MaxSlots)
}

func (e *eventUserService) validateUserExists(userId string) *rest_err.RestErr {
	if _, err := e.userService.FindById(userId); err != nil {
		return rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			[]rest_err.Causes{{
				Field:   "user_id",
				Message: "user_id is not valid, not found this user",
			}})
	}
	return nil
}
