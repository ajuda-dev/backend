package dto

import (
	"time"

	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type RegisterEventDto struct {
	Id          string    `json:"id"`
	OwnerId     string    `json:"owner_id"`
	CommunityId *string   `json:"community_id"`
	AddressId   *string   `json:"address_id"`
	Category    string    `json:"category"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"start_at"`
	DurationMin int       `json:"duration_min"`
	MeetingLink string    `json:"meeting_link"`
	MaxSlots    *int      `json:"max_slots"`
	CreatorRole string    `json:"creator_role"`
	Status      string    `json:"status"`
	Comment     string    `json:"comment,omitempty"`
}

func (r *RegisterEventDto) ToDomain() *eventdomain.EventDomain {
	var community *communitydomain.CommunityDomain
	if r.CommunityId != nil {
		community = &communitydomain.CommunityDomain{Id: *r.CommunityId}
	}
	var address *addressdomain.AddressDomain
	if r.AddressId != nil {
		address = &addressdomain.AddressDomain{Id: *r.AddressId}
	}
	return &eventdomain.EventDomain{
		Owner:       userdomain.UserDomain{Id: r.OwnerId},
		Community:   community,
		Address:     address,
		Category:    r.Category,
		Type:        r.Type,
		Title:       r.Title,
		Description: r.Description,
		StartAt:     r.StartAt,
		DurationMin: r.DurationMin,
		MeetingLink: r.MeetingLink,
		MaxSlots:    r.MaxSlots,
		CreatorRole: r.CreatorRole,
	}
}

func (r RegisterEventDto) FromDomain(event *eventdomain.EventDomain) interface{} {
	dtoEvent := &RegisterEventDto{
		Id:          event.Id,
		OwnerId:     event.Owner.Id,
		Category:    event.Category,
		Type:        event.Type,
		Title:       event.Title,
		Description: event.Description,
		StartAt:     event.StartAt,
		DurationMin: event.DurationMin,
		MeetingLink: event.MeetingLink,
		MaxSlots:    event.MaxSlots,
		CreatorRole: event.CreatorRole,
		Status:      event.Status,
		Comment:     event.Comment,
	}
	if event.Community != nil {
		dtoEvent.CommunityId = &event.Community.Id
	}
	if event.Address != nil {
		dtoEvent.AddressId = &event.Address.Id
	}
	return dtoEvent
}
