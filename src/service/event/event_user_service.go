package event

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	eventvalidator "github.com/ajuda-dev/backend/src/service/event/validator"
	"github.com/ajuda-dev/backend/src/service/identity"
)

type EventUserService interface {
	JoinEvent(eventId string, requesterId string) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	AddParticipant(eventId string, requesterId string, targetUserId string, role string) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	GetParticipants(eventId string, status string, requesterId string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr)
	UpdateParticipantStatus(eventId string, userId string, requesterId string, status string) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	CancelParticipation(eventId string, userId string, requesterId string) (*eventdomain.EventUserDomain, *rest_err.RestErr)
}

type eventUserService struct {
	userService         identity.UserService
	eventService        EventService
	eventUserRepository eventrepo.EventUserRepository
	eventUserValidator  eventvalidator.EventUserValidator
}

func NewEventUserService(
	userService identity.UserService,
	eventService EventService,
	eventUserRepository eventrepo.EventUserRepository,
	eventUserValidator eventvalidator.EventUserValidator) EventUserService {
	return &eventUserService{
		userService:         userService,
		eventService:        eventService,
		eventUserRepository: eventUserRepository,
		eventUserValidator:  eventUserValidator,
	}
}

func (e *eventUserService) JoinEvent(eventId string, requesterId string) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if !isEventApproved(event) {
		return nil, eventNotApprovedError()
	}
	eventUser := &eventdomain.EventUserDomain{
		EventId: eventId,
		UserId:  requester.Id,
		Role:    eventdomain.RoleAttendee,
		Status:  eventdomain.StatusConfirmed,
	}
	if err := e.eventUserValidator.ValidateJoin(*eventUser, event.Category); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) AddParticipant(eventId string, requesterId string, targetUserId string, role string) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
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
	if event.Category == eventdomain.CategoryMentoring {
		creatorRole, err = e.creatorRole(event)
		if err != nil {
			return nil, err
		}
	}
	eventUser := &eventdomain.EventUserDomain{
		EventId: eventId,
		UserId:  targetUserId,
		Role:    role,
	}
	if err := e.eventUserValidator.ValidateAddParticipant(*eventUser, event.Category, creatorRole); err != nil {
		return nil, err
	}
	if event.Category == eventdomain.CategoryMentoring {
		eventUser.Status = eventdomain.StatusRequested
	} else {
		eventUser.Status = eventdomain.StatusConfirmed
	}
	if err := e.validateUserExists(eventUser.UserId); err != nil {
		return nil, err
	}
	return e.eventUserRepository.CreateOrUpdate(eventUser, event.MaxSlots)
}

func (e *eventUserService) GetParticipants(eventId string, status string, requesterId string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventService.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if !isEventApproved(event) && !canManageEvent(requester, event) {
		return nil, rest_err.NewNotFoundError("event not found")
	}
	participants, err := e.eventUserRepository.FindByEvent(eventId, status)
	if err != nil {
		return nil, err
	}
	for _, participant := range participants {
		identity.ApplyVisibilityFilter(participant.User, requester)
	}
	return participants, nil
}

func (e *eventUserService) UpdateParticipantStatus(eventId string, userId string, requesterId string, status string) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
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
	if status == eventdomain.StatusConfirmed && !isEventApproved(event) {
		return nil, eventNotApprovedError()
	}
	if err := e.eventUserValidator.ValidateUpdateParticipantStatus(status); err != nil {
		return nil, err
	}
	return e.eventUserRepository.UpdateStatus(eventId, userId, status, event.MaxSlots)
}

func (e *eventUserService) CancelParticipation(eventId string, userId string, requesterId string) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
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
	if event.Category == eventdomain.CategoryMentoring && userId == event.Owner.Id {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid participation data",
			[]rest_err.Causes{{
				Field:   "user_id",
				Message: "the creator cannot leave the event; cancel the event instead",
			}})
	}
	return e.eventUserRepository.UpdateStatus(eventId, userId, eventdomain.StatusCancelled, event.MaxSlots)
}

func (e *eventUserService) creatorRole(event *eventdomain.EventDomain) (string, *rest_err.RestErr) {
	participants, err := e.eventUserRepository.FindByEvent(event.Id, "")
	if err != nil {
		return "", err
	}
	for _, participant := range participants {
		if participant.UserId == event.Owner.Id {
			return participant.Role, nil
		}
	}
	return eventdomain.RoleMentor, nil
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
