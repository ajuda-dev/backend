package entity

import (
	"time"

	addressentity "github.com/ajuda-dev/backend/src/data/address/entity"
	communityentity "github.com/ajuda-dev/backend/src/data/community/entity"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	"gorm.io/gorm"
)

type EventEntity struct {
	Id          string `gorm:"primaryKey;type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Category    string         `gorm:"type:varchar(30);not null;index"`
	Type        string         `gorm:"type:varchar(30);not null;index"`
	Title       string         `gorm:"not null"`
	Description string
	StartAt     time.Time `gorm:"not null;index:idx_events_community_start,priority:2"`
	DurationMin int       `gorm:"not null;default:60"`
	OwnerId     string    `gorm:"type:uuid;not null;index"`
	CommunityId *string   `gorm:"type:uuid;index:idx_events_community_start,priority:1"`
	AddressId   *string   `gorm:"type:uuid;index"`
	MeetingLink string
	MaxSlots    *int
	Status      string `gorm:"type:varchar(20);not null;default:'PENDING';index"`

	Community *communityentity.CommunityEntity `gorm:"foreignKey:CommunityId;references:Id;constraint:OnDelete:RESTRICT"`
	Address   *addressentity.AddressEntity     `gorm:"foreignKey:AddressId;references:Id;constraint:OnDelete:SET NULL"`
	Owner     userentity.UserEntity            `gorm:"foreignKey:OwnerId;references:Id;constraint:OnDelete:RESTRICT"`
}

func (EventEntity) TableName() string {
	return "events"
}

func (e *EventEntity) FromDomain(event eventdomain.EventDomain) *EventEntity {
	eventEntity := &EventEntity{
		Id:          event.Id,
		Category:    event.Category,
		Type:        event.Type,
		Title:       event.Title,
		Description: event.Description,
		StartAt:     event.StartAt,
		DurationMin: event.DurationMin,
		OwnerId:     event.Owner.Id,
		MeetingLink: event.MeetingLink,
		MaxSlots:    event.MaxSlots,
		Status:      event.Status,
	}
	if event.Community != nil {
		communityId := event.Community.Id
		eventEntity.CommunityId = &communityId
	}
	if event.Address != nil {
		addressId := event.Address.Id
		eventEntity.AddressId = &addressId
	}
	return eventEntity
}

func (e EventEntity) ToDomain() *eventdomain.EventDomain {
	event := &eventdomain.EventDomain{
		Id:          e.Id,
		Category:    e.Category,
		Type:        e.Type,
		Title:       e.Title,
		Description: e.Description,
		StartAt:     e.StartAt,
		DurationMin: e.DurationMin,
		Owner:       *e.Owner.ToDomainUser(),
		MeetingLink: e.MeetingLink,
		MaxSlots:    e.MaxSlots,
		Status:      e.Status,
	}
	if e.Community != nil {
		event.Community = e.Community.ToDomain()
	}
	if e.Address != nil {
		event.Address = e.Address.ToDomainAddress()
	}
	return event
}

func ToEventDomainList(entities []EventEntity) []*eventdomain.EventDomain {
	var domains []*eventdomain.EventDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
