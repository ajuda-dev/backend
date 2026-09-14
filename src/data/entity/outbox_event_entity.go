package entity

import (
	"encoding/json"
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type OutboxEventEntity struct {
	Id        int64           `gorm:"primaryKey;autoIncrement"`
	Type      string          `gorm:"type:varchar(80);not null"`
	UserId    string          `gorm:"type:uuid;not null;index"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null"`
	Status    string          `gorm:"type:varchar(20);not null;index:idx_outbox_events_status_created,priority:1"`
	CreatedAt time.Time       `gorm:"not null;index:idx_outbox_events_status_created,priority:2"`
	UpdatedAt time.Time       `gorm:"not null"`

	User UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:RESTRICT"`
}

func (OutboxEventEntity) TableName() string {
	return "outbox_events"
}

func (e *OutboxEventEntity) FromDomain(d domain.OutboxEventDomain) *OutboxEventEntity {
	return &OutboxEventEntity{
		Id:      d.Id,
		Type:    d.Type,
		UserId:  d.UserId,
		Payload: d.Payload,
		Status:  d.Status,
	}
}

func (e OutboxEventEntity) ToDomain() *domain.OutboxEventDomain {
	return &domain.OutboxEventDomain{
		Id:        e.Id,
		Type:      e.Type,
		UserId:    e.UserId,
		Payload:   e.Payload,
		Status:    e.Status,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
