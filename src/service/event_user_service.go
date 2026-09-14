package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type EventUserService interface {
	JoinEvent(eventId string, requesterId string) (*domain.EventUserDomain, *rest_err.RestErr)
	AddParticipant(eventId string, requesterId string, targetUserId string, role string) (*domain.EventUserDomain, *rest_err.RestErr)
	GetParticipants(eventId string, status string, requesterId string) ([]*domain.EventUserDomain, *rest_err.RestErr)
	UpdateParticipantStatus(eventId string, userId string, requesterId string, status string) (*domain.EventUserDomain, *rest_err.RestErr)
	CancelParticipation(eventId string, userId string, requesterId string) (*domain.EventUserDomain, *rest_err.RestErr)
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

func (e *eventUserService) JoinEvent(eventId string, requesterId string) (*domain.EventUserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	eventUser := &domain.EventUserDomain{
		EventId: eventId,
		UserId:  requester.Id,
		Role:    domain.RoleAttendee,
		Status:  domain.StatusConfirmed,
	}
	if err := e.eventUserValidator.ValidateJoin(*eventUser, event.Category); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) AddParticipant(eventId string, requesterId string, targetUserId string, role string) (*domain.EventUserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if !canManageEvent(requester, event) {
		return nil, forbiddenManageEvent()
	}
	creatorRole := ""
	if event.Category == domain.CategoryMentoring {
		creatorRole, err = e.creatorRole(event)
		if err != nil {
			return nil, err
		}
	}
	eventUser := &domain.EventUserDomain{
		EventId: eventId,
		UserId:  targetUserId,
		Role:    role,
	}
	if err := e.eventUserValidator.ValidateAddParticipant(*eventUser, event.Category, creatorRole); err != nil {
		return nil, err
	}
	if event.Category == domain.CategoryMentoring {
		eventUser.Status = domain.StatusRequested
	} else {
		eventUser.Status = domain.StatusConfirmed
	}
	if err := e.validateUserExists(eventUser.UserId); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) GetParticipants(eventId string, status string, requesterId string) ([]*domain.EventUserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if _, err := e.eventService.GetEventById(eventId); err != nil {
		return nil, err
	}
	participants, err := e.eventUserRepository.FindByEvent(eventId, status)
	if err != nil {
		return nil, err
	}
	for _, participant := range participants {
		applyVisibilityFilter(participant.User, requester)
	}
	return participants, nil
}

func (e *eventUserService) UpdateParticipantStatus(eventId string, userId string, requesterId string, status string) (*domain.EventUserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if userId != requester.Id {
		return nil, rest_err.NewForbiddenError("only the invited user can accept or reject this invitation")
	}
	if err := e.eventUserValidator.ValidateUpdateParticipantStatus(status); err != nil {
		return nil, err
	}
	return e.eventUserRepository.UpdateStatus(eventId, userId, status, event.MaxSlots)
}

func (e *eventUserService) CancelParticipation(eventId string, userId string, requesterId string) (*domain.EventUserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if userId != requester.Id && !canManageEvent(requester, event) {
		return nil, forbiddenManageEvent()
	}
	if event.Category == domain.CategoryMentoring && userId == event.Owner.Id {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			[]rest_err.Causes{{
				Field:   "user_id",
				Message: "the creator cannot leave the event; cancel the event instead",
			}})
	}
	return e.eventUserRepository.UpdateStatus(eventId, userId, domain.StatusCancelled, event.MaxSlots)
}

func (e *eventUserService) creatorRole(event *domain.EventDomain) (string, *rest_err.RestErr) {
	participants, err := e.eventUserRepository.FindByEvent(event.Id, "")
	if err != nil {
		return "", err
	}
	for _, participant := range participants {
		if participant.UserId == event.Owner.Id {
			return participant.Role, nil
		}
	}
	return domain.RoleMentor, nil
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
