package entity

import (
	"encoding/json"
	"time"

	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
)

type OutboxEventEntity struct {
	Id        int64           `gorm:"primaryKey;autoIncrement"`
	Type      string          `gorm:"type:varchar(80);not null"`
	UserId    string          `gorm:"type:uuid;not null;index;index:idx_outbox_events_user_read_created,priority:1"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null"`
	Status    string          `gorm:"type:varchar(20);not null;index:idx_outbox_events_status_created,priority:1"`
	ReadAt    *time.Time      `gorm:"index:idx_outbox_events_user_read_created,priority:2"`
	CreatedAt time.Time       `gorm:"not null;index:idx_outbox_events_status_created,priority:2;index:idx_outbox_events_user_read_created,priority:3"`
	UpdatedAt time.Time       `gorm:"not null"`

	User userentity.UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:RESTRICT"`
}

func (OutboxEventEntity) TableName() string {
	return "outbox_events"
}

func (e *OutboxEventEntity) FromDomain(d notificationdomain.OutboxEventDomain) *OutboxEventEntity {
	return &OutboxEventEntity{
		Id:      d.Id,
		Type:    d.Type,
		UserId:  d.UserId,
		Payload: d.Payload,
		Status:  d.Status,
	}
}

func (e OutboxEventEntity) ToDomain() *notificationdomain.OutboxEventDomain {
	readAt := ""
	if e.ReadAt != nil {
		readAt = e.ReadAt.UTC().Format(time.RFC3339Nano)
	}
	return &notificationdomain.OutboxEventDomain{
		Id:        e.Id,
		Type:      e.Type,
		UserId:    e.UserId,
		Payload:   e.Payload,
		Status:    e.Status,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
		ReadAt:    readAt,
	}
}
