package dto

import (
	"time"

	addressdto "github.com/ajuda-dev/backend/src/controller/address/dto"
	communitydto "github.com/ajuda-dev/backend/src/controller/community/dto"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
)

type PageableEventDto struct {
	HasNext bool       `json:"has_next"`
	Data    []EventDto `json:"data"`
}

type EventDto struct {
	Id          string                     `json:"id"`
	Category    string                     `json:"category"`
	Type        string                     `json:"type"`
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	StartAt     time.Time                  `json:"start_at"`
	DurationMin int                        `json:"duration_min"`
	MeetingLink string                     `json:"meeting_link"`
	MaxSlots    *int                       `json:"max_slots"`
	Status      string                     `json:"status"`
	Comment     string                     `json:"comment,omitempty"`
	Owner       *userdto.UserDtoOut        `json:"owner"`
	Community   *communitydto.CommunityDto `json:"community"`
	Address     *addressdto.AddressDto     `json:"address"`
}

func (e EventDto) FromDomain(event *eventdomain.EventDomain) EventDto {
	dtoEvent := EventDto{
		Id:          event.Id,
		Category:    event.Category,
		Type:        event.Type,
		Title:       event.Title,
		Description: event.Description,
		StartAt:     event.StartAt,
		DurationMin: event.DurationMin,
		MeetingLink: event.MeetingLink,
		MaxSlots:    event.MaxSlots,
		Status:      event.Status,
		Comment:     event.Comment,
		Owner:       (&userdto.UserDtoOut{}).FromDomainUser(&event.Owner),
	}
	if event.Community != nil {
		dtoEvent.Community = toCommunityDto(event.Community)
	}
	if event.Address != nil {
		dtoEvent.Address = (&addressdto.AddressDto{}).FromDomain(event.Address)
	}
	return dtoEvent
}

func toCommunityDto(community *communitydomain.CommunityDomain) *communitydto.CommunityDto {
	return &communitydto.CommunityDto{
		Id:          community.Id,
		Name:        community.Name,
		Description: community.Description,
		Address:     (&addressdto.AddressDto{}).FromDomain(&community.Address),
		Owner:       (&userdto.UserDtoOut{}).FromDomainUser(&community.Owner),
	}
}

func (pe PageableEventDto) FromDomain(page eventdomain.PageableEvent) *PageableEventDto {
	events := make([]EventDto, len(page.Data))
	for i, e := range page.Data {
		events[i] = EventDto{}.FromDomain(e)
	}
	return &PageableEventDto{
		HasNext: page.HasNext,
		Data:    events,
	}
}
