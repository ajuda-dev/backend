package dto

import (
	"encoding/json"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type PageableNotificationDto struct {
	HasNext bool                 `json:"has_next"`
	Data    []NotificationDtoOut `json:"data"`
}

type NotificationDtoOut struct {
	Id        int64           `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload" swaggertype:"object"`
	CreatedAt string          `json:"created_at"`
	ReadAt    *string         `json:"read_at"`
}

func (p PageableNotificationDto) FromDomain(page domain.PageableOutboxEvent) *PageableNotificationDto {
	data := make([]NotificationDtoOut, len(page.Data))
	for i, item := range page.Data {
		data[i] = NotificationDtoOut{}.FromDomain(item)
	}
	return &PageableNotificationDto{HasNext: page.HasNext, Data: data}
}

func (n NotificationDtoOut) FromDomain(event domain.OutboxEventDomain) NotificationDtoOut {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}
	out := NotificationDtoOut{
		Id:        event.Id,
		Type:      event.Type,
		Payload:   payload,
		CreatedAt: event.CreatedAt,
	}
	if event.ReadAt != "" {
		readAt := event.ReadAt
		out.ReadAt = &readAt
	}
	return out
}
