package event

import (
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	"github.com/ajuda-dev/backend/src/service/address"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	eventvalidator "github.com/ajuda-dev/backend/src/service/event/validator"
	"github.com/ajuda-dev/backend/src/service/identity"
)

type EventService interface {
	CreateEvent(event *eventdomain.EventDomain) (*eventdomain.EventDomain, *rest_err.RestErr)
	GetEventById(id string) (*eventdomain.EventDomain, *rest_err.RestErr)
	GetEventDetail(id string, requesterId string) (*eventdomain.EventDomain, *rest_err.RestErr)
	GetAll(filter eventrepo.EventFilter, page int, limit int, requesterId string) (*eventdomain.PageableEvent, *rest_err.RestErr)
	Reschedule(id string, requesterId string, startAt time.Time, comment string) (*eventdomain.EventDomain, *rest_err.RestErr)
	DeleteEventById(id string, requesterId string, comment string) *rest_err.RestErr
	UpdateApproval(id string, requesterId string, status string) (*eventdomain.EventDomain, *rest_err.RestErr)
}

type eventService struct {
	eventRepository         eventrepo.EventRepository
	eventUserRepository     eventrepo.EventUserRepository
	eventValidator          eventvalidator.EventValidator
	userService             identity.UserService
	addressService          address.AddressService
	communityRepository     communityrepo.CommunityRepository
	communityUserRepository communityrepo.CommunityUserRepository
}

func NewEventService(
	userService identity.UserService,
	addressService address.AddressService,
	communityRepository communityrepo.CommunityRepository,
	eventRepository eventrepo.EventRepository,
	eventUserRepository eventrepo.EventUserRepository,
	communityUserRepository communityrepo.CommunityUserRepository,
	eventValidator eventvalidator.EventValidator) EventService {
	return &eventService{
		userService:             userService,
		addressService:          addressService,
		communityRepository:     communityRepository,
		eventRepository:         eventRepository,
		eventUserRepository:     eventUserRepository,
		communityUserRepository: communityUserRepository,
		eventValidator:          eventValidator,
	}
}

func (e *eventService) CreateEvent(event *eventdomain.EventDomain) (*eventdomain.EventDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, event.Owner.Id)
	if err != nil {
		return &eventdomain.EventDomain{}, err
	}
	if err := identity.RequireVerifiedEmail(requester); err != nil {
		return &eventdomain.EventDomain{}, err
	}
	event.Owner = *requester
	event.Status = eventdomain.EventStatusApproved
	if event.Category == eventdomain.CategoryMentoring {
		maxSlots := 2
		event.MaxSlots = &maxSlots
		if event.CreatorRole == "" {
			event.CreatorRole = eventdomain.RoleMentor
		}
	}

	if err := e.eventValidator.ValidatorRegisterEvent(*event); err != nil {
		return &eventdomain.EventDomain{}, err
	}

	if event.Address != nil {
		address, err_a := e.addressService.GetAddressById(event.Address.Id)
		if err_a != nil {
			if err_a.Code == rest_err.NOT_FOUND {
				return &eventdomain.EventDomain{},
					rest_err.NewBadRequestValidationError("Invalid event data", []rest_err.Causes{
						{
							Field:   "address_id",
							Message: "address_id is not valid, not found this address",
						},
					})
			}
			return &eventdomain.EventDomain{}, err_a
		}
		event.Address = address
	}

	if event.Community != nil {
		community, err_c := e.communityRepository.FindById(event.Community.Id)
		if err_c != nil {
			if err_c.Code == rest_err.NOT_FOUND {
				return &eventdomain.EventDomain{},
					rest_err.NewBadRequestValidationError("Invalid event data", []rest_err.Causes{
						{
							Field:   "community_id",
							Message: "community_id is not valid, not found this community",
						},
					})
			}
			return &eventdomain.EventDomain{}, err_c
		}
		event.Community = community
		if community.Owner.Id != requester.Id {
			member, memberErr := e.communityUserRepository.ExistsByCommunityAndUser(community.Id, requester.Id)
			if memberErr != nil {
				return &eventdomain.EventDomain{}, memberErr
			}
			if !member && !isStaff(requester) {
				return &eventdomain.EventDomain{}, rest_err.NewForbiddenError("only community members can create events for this community")
			}
			event.Status = eventdomain.EventStatusPending
		}
	}

	createdEvent, createErr := e.eventRepository.CreateEvent(event)
	if createErr != nil {
		return &eventdomain.EventDomain{}, createErr
	}

	if event.Category == eventdomain.CategoryMentoring {
		if _, mentorErr := e.eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
			EventId: createdEvent.Id,
			UserId:  createdEvent.Owner.Id,
			Role:    event.CreatorRole,
			Status:  eventdomain.StatusConfirmed,
		}, createdEvent.MaxSlots); mentorErr != nil {
			return &eventdomain.EventDomain{}, mentorErr
		}
	}

	return createdEvent, nil
}

func (e *eventService) GetEventById(id string) (*eventdomain.EventDomain, *rest_err.RestErr) {
	return e.eventRepository.FindById(id)
}

func (e *eventService) GetEventDetail(id string, requesterId string) (*eventdomain.EventDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventRepository.FindById(id)
	if err != nil {
		return nil, err
	}
	if !isEventVisible(requester, event) {
		return nil, rest_err.NewNotFoundError("event not found")
	}
	identity.ApplyVisibilityFilter(&event.Owner, requester)
	if event.Community != nil {
		identity.ApplyVisibilityFilter(&event.Community.Owner, requester)
	}
	return event, nil
}

func (e *eventService) GetAll(filter eventrepo.EventFilter, page int, limit int, requesterId string) (*eventdomain.PageableEvent, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if filter.UserId != "" && filter.UserId != requester.Id && !isStaff(requester) {
		return nil, rest_err.NewForbiddenError("only moderators and admins can read another user's agenda")
	}
	if filter.ApprovalStatus != "" {
		if !isStaff(requester) && !isCommunityOwner(requester, filter.CommunityId, e.communityRepository) {
			return nil, rest_err.NewForbiddenError("only the community owner can filter events by approval status")
		}
	}
	filter.RequesterId = requester.Id
	pageable, err := e.eventRepository.FindAll(filter, page, limit)
	if err != nil {
		return nil, err
	}
	for _, event := range pageable.Data {
		identity.ApplyVisibilityFilter(&event.Owner, requester)
		if event.Community != nil {
			identity.ApplyVisibilityFilter(&event.Community.Owner, requester)
		}
	}
	return pageable, nil
}

func (e *eventService) Reschedule(id string, requesterId string, startAt time.Time, comment string) (*eventdomain.EventDomain, *rest_err.RestErr) {
	comment = strings.TrimSpace(comment)
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventRepository.FindById(id)
	if err != nil {
		return nil, err
	}
	isMentoringParticipant := false
	if event.Category == eventdomain.CategoryMentoring && !canManageEvent(requester, event) {
		participants, partErr := e.eventUserRepository.FindByEvent(id, "")
		if partErr != nil {
			return nil, partErr
		}
		isMentoringParticipant = isActiveMentoringParticipant(participants, requester.Id)
	}
	if !canRescheduleEvent(requester, event, isMentoringParticipant) {
		if event.Category == eventdomain.CategoryMentoring {
			return nil, forbiddenRescheduleEvent()
		}
		return nil, forbiddenManageEvent()
	}
	if err := e.eventValidator.ValidateReschedule(startAt, comment); err != nil {
		return nil, err
	}
	return e.eventRepository.Reschedule(id, startAt, comment, requester.Id)
}

func (e *eventService) DeleteEventById(id string, requesterId string, comment string) *rest_err.RestErr {
	comment = strings.TrimSpace(comment)
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return err
	}
	event, err := e.eventRepository.FindById(id)
	if err != nil {
		return err
	}
	if !canManageEvent(requester, event) {
		return forbiddenManageEvent()
	}
	if err := e.eventValidator.ValidateCancelComment(comment); err != nil {
		return err
	}
	return e.eventRepository.SoftDeleteById(id, comment)
}

func isValidEventStatusTransition(current string, target string) bool {
	switch current {
	case eventdomain.EventStatusPending:
		return target == eventdomain.EventStatusApproved || target == eventdomain.EventStatusRejected
	case eventdomain.EventStatusRejected:
		return target == eventdomain.EventStatusApproved
	}
	return false
}

func (e *eventService) UpdateApproval(id string, requesterId string, status string) (*eventdomain.EventDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventRepository.FindById(id)
	if err != nil {
		return nil, err
	}
	if !canApproveEvent(requester, event) {
		return nil, forbiddenApproveEvent()
	}
	if !isValidEventStatusTransition(event.Status, status) {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid event data",
			[]rest_err.Causes{{
				Field:   "status",
				Message: "invalid status transition from " + event.Status + " to " + status,
			}})
	}
	return e.eventRepository.UpdateApprovalStatus(id, event.Status, status, requester.Id)
}
