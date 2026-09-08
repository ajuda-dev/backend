package dto

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type PageableEventDto struct {
	HasNext bool       `json:"has_next"`
	Data    []EventDto `json:"data"`
}

type EventDto struct {
	Id          string        `json:"id"`
	Category    string        `json:"category"`
	Type        string        `json:"type"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	StartAt     time.Time     `json:"start_at"`
	DurationMin int           `json:"duration_min"`
	MeetingLink string        `json:"meeting_link"`
	MaxSlots    *int          `json:"max_slots"`
	Owner       *UserDtoOut   `json:"owner"`
	Community   *CommunityDto `json:"community"`
	Address     *AddressDto   `json:"address"`
}

func (e EventDto) FromDomain(event *domain.EventDomain) EventDto {
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
		Owner:       (&UserDtoOut{}).FromDomainUser(&event.Owner),
	}
	if event.Community != nil {
		dtoEvent.Community = toCommunityDto(event.Community)
	}
	if event.Address != nil {
		dtoEvent.Address = (&AddressDto{}).FromDomain(event.Address)
	}
	return dtoEvent
}

func toCommunityDto(community *domain.CommunityDomain) *CommunityDto {
	return &CommunityDto{
		Id:          community.Id,
		Name:        community.Name,
		Description: community.Description,
		Address:     (&AddressDto{}).FromDomain(&community.Address),
		Owner:       (&UserDtoOut{}).FromDomainUser(&community.Owner),
	}
}

func (pe PageableEventDto) FromDomain(page domain.PageableEvent) *PageableEventDto {
	events := make([]EventDto, len(page.Data))
	for i, e := range page.Data {
		events[i] = EventDto{}.FromDomain(e)
	}
	return &PageableEventDto{
		HasNext: page.HasNext,
		Data:    events,
	}
}
