package job

import (
	"encoding/json"

	"github.com/ajuda-dev/backend/src/service/notification"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
)

type SSEOutboxHandler struct {
	hub notification.NotificationHub
}

func NewSSEOutboxHandler(hub notification.NotificationHub) OutboxHandler {
	return &SSEOutboxHandler{hub: hub}
}

func (h *SSEOutboxHandler) Handle(event notificationdomain.OutboxEventDomain) error {
	body, err := json.Marshal(map[string]any{
		"id":      event.Id,
		"type":    event.Type,
		"payload": json.RawMessage(event.Payload),
	})
	if err != nil {
		return err
	}
	if h.hub != nil {
		h.hub.Publish(event.UserId, body)
	}
	return nil
}
