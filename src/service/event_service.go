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
	GetAll(filter repository.EventFilter, page int, limit int) (*domain.PageableEvent, *rest_err.RestErr)
	DeleteEventById(id string) *rest_err.RestErr
}

type eventService struct {
	eventRepository     repository.EventRepository
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
	eventValidator validator.EventValidator) EventService {
	return &eventService{
		userService:         userService,
		addressService:      addressService,
		communityRepository: communityRepository,
		eventRepository:     eventRepository,
		eventValidator:      eventValidator,
	}
}

func (e *eventService) CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr) {
	err := e.eventValidator.ValidatorRegisterEvent(*event)
	if err != nil {
		return &domain.EventDomain{}, err
	}

	user, err_u := e.userService.FindById(event.Owner.Id)
	if err_u != nil {
		return &domain.EventDomain{}, err_u
	}
	event.Owner = *user

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

	return e.eventRepository.CreateEvent(event)
}

func (e *eventService) GetEventById(id string) (*domain.EventDomain, *rest_err.RestErr) {
	return e.eventRepository.FindById(id)
}

func (e *eventService) GetAll(filter repository.EventFilter, page int, limit int) (*domain.PageableEvent, *rest_err.RestErr) {
	return e.eventRepository.FindAll(filter, page, limit)
}

func (e *eventService) DeleteEventById(id string) *rest_err.RestErr {
	return e.eventRepository.SoftDeleteById(id)
}
