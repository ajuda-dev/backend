package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type EventUserEntity struct {
	Id        string `gorm:"primaryKey;type:uuid"`
	EventId   string `gorm:"type:uuid;not null;uniqueIndex:idx_event_users_event_user,priority:1;index"`
	UserId    string `gorm:"type:uuid;not null;uniqueIndex:idx_event_users_event_user,priority:2;index:idx_event_users_user_status,priority:1"`
	Role      string `gorm:"type:varchar(20);not null"`                                              // HOST | MENTOR | MENTEE | SPEAKER | ATTENDEE
	Status    string `gorm:"type:varchar(20);not null;index:idx_event_users_user_status,priority:2"` // REQUESTED | CONFIRMED | REJECTED | CANCELLED
	CreatedAt time.Time
	UpdatedAt time.Time

	Event EventEntity `gorm:"foreignKey:EventId;references:Id;constraint:OnDelete:CASCADE"`
	User  UserEntity  `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:RESTRICT"`
}

func (EventUserEntity) TableName() string {
	return "event_users"
}

func (e *EventUserEntity) FromDomain(eventUser domain.EventUserDomain) *EventUserEntity {
	return &EventUserEntity{
		Id:      eventUser.Id,
		EventId: eventUser.EventId,
		UserId:  eventUser.UserId,
		Role:    eventUser.Role,
		Status:  eventUser.Status,
	}
}

func (e EventUserEntity) ToDomain() *domain.EventUserDomain {
	eventUser := &domain.EventUserDomain{
		Id:      e.Id,
		EventId: e.EventId,
		UserId:  e.UserId,
		Role:    e.Role,
		Status:  e.Status,
	}
	if e.User.Id != "" {
		eventUser.User = e.User.ToDomainUser()
	}
	return eventUser
}

func ToEventUserDomainList(entities []EventUserEntity) []*domain.EventUserDomain {
	var domains []*domain.EventUserDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
