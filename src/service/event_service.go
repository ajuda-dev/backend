package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type EventService interface {
	CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr)
	GetEventById(id string) (*domain.EventDomain, *rest_err.RestErr)
	GetEventDetail(id string, requesterId string) (*domain.EventDomain, *rest_err.RestErr)
	GetAll(filter repository.EventFilter, page int, limit int, requesterId string) (*domain.PageableEvent, *rest_err.RestErr)
	DeleteEventById(id string, requesterId string) *rest_err.RestErr
}

type eventService struct {
	eventRepository     repository.EventRepository
	eventUserRepository repository.EventUserRepository
	eventValidator      validator.EventValidator
	userService         UserService
	addressService      AddressService
	communityRepository repository.CommunityRepository
}

func NewEventService(
	userService UserService,
	addressService AddressService,
	communityRepository repository.CommunityRepository,
	eventRepository repository.EventRepository,
	eventUserRepository repository.EventUserRepository,
	eventValidator validator.EventValidator) EventService {
	return &eventService{
		userService:         userService,
		addressService:      addressService,
		communityRepository: communityRepository,
		eventRepository:     eventRepository,
		eventUserRepository: eventUserRepository,
		eventValidator:      eventValidator,
	}
}

func (e *eventService) CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, event.Owner.Id)
	if err != nil {
		return &domain.EventDomain{}, err
	}
	event.Owner = *requester
	if event.Category == domain.CategoryMentoring {
		maxSlots := 2
		event.MaxSlots = &maxSlots
		if event.CreatorRole == "" {
			event.CreatorRole = domain.RoleMentor
		}
	}

	if err := e.eventValidator.ValidatorRegisterEvent(*event); err != nil {
		return &domain.EventDomain{}, err
	}

	if event.Address != nil {
		address, err_a := e.addressService.GetAddressById(event.Address.Id)
		if err_a != nil {
			if err_a.Code == rest_err.NOT_FOUND {
				return &domain.EventDomain{},
					rest_err.NewBadRequestValidationError("Invalid event data", []rest_err.Causes{
						{
							Field:   "address_id",
							Message: "address_id is not valid, not found this address",
						},
					})
			}
			return &domain.EventDomain{}, err_a
		}
		event.Address = address
	}

	if event.Community != nil {
		community, err_c := e.communityRepository.FindById(event.Community.Id)
		if err_c != nil {
			if err_c.Code == rest_err.NOT_FOUND {
				return &domain.EventDomain{},
					rest_err.NewBadRequestValidationError("Invalid event data", []rest_err.Causes{
						{
							Field:   "community_id",
							Message: "community_id is not valid, not found this community",
						},
					})
			}
			return &domain.EventDomain{}, err_c
		}
		event.Community = community
	}

	createdEvent, createErr := e.eventRepository.CreateEvent(event)
	if createErr != nil {
		return &domain.EventDomain{}, createErr
	}

	if event.Category == domain.CategoryMentoring {
		if _, mentorErr := e.eventUserRepository.CreateOrUpdate(&domain.EventUserDomain{
			EventId: createdEvent.Id,
			UserId:  createdEvent.Owner.Id,
			Role:    event.CreatorRole,
			Status:  domain.StatusConfirmed,
		}, createdEvent.MaxSlots); mentorErr != nil {
			return &domain.EventDomain{}, mentorErr
		}
	}

	return createdEvent, nil
}

func (e *eventService) GetEventById(id string) (*domain.EventDomain, *rest_err.RestErr) {
	return e.eventRepository.FindById(id)
}

func (e *eventService) GetEventDetail(id string, requesterId string) (*domain.EventDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	event, err := e.eventRepository.FindById(id)
	if err != nil {
		return nil, err
	}
	applyVisibilityFilter(&event.Owner, requester)
	if event.Community != nil {
		applyVisibilityFilter(&event.Community.Owner, requester)
	}
	return event, nil
}

func (e *eventService) GetAll(filter repository.EventFilter, page int, limit int, requesterId string) (*domain.PageableEvent, *rest_err.RestErr) {
	requester, err := authenticatedUser(e.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if filter.UserId != "" && filter.UserId != requester.Id && !isStaff(requester) {
		return nil, rest_err.NewForbiddenError("only moderators and admins can read another user's agenda")
	}
	pageable, err := e.eventRepository.FindAll(filter, page, limit)
	if err != nil {
		return nil, err
	}
	for _, event := range pageable.Data {
		applyVisibilityFilter(&event.Owner, requester)
		if event.Community != nil {
			applyVisibilityFilter(&event.Community.Owner, requester)
		}
	}
	return pageable, nil
}

func (e *eventService) DeleteEventById(id string, requesterId string) *rest_err.RestErr {
	requester, err := authenticatedUser(e.userService, requesterId)
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
	return e.eventRepository.SoftDeleteById(id)
}
