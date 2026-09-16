package job

import (
	"fmt"

	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
)

type dispatchingOutboxHandler struct {
	inbox          OutboxHandler
	createdAccount OutboxHandler
}

func NewDispatchingOutboxHandler(inbox, createdAccount OutboxHandler) OutboxHandler {
	return &dispatchingOutboxHandler{
		inbox:          inbox,
		createdAccount: createdAccount,
	}
}

func (h *dispatchingOutboxHandler) Handle(event notificationdomain.OutboxEventDomain) error {
	if notificationdomain.IsInboxNotificationType(event.Type) {
		if h.inbox == nil {
			return fmt.Errorf("inbox outbox handler is not configured")
		}
		return h.inbox.Handle(event)
	}
	if event.Type == notificationdomain.OutboxTypeCreatedAccount {
		if h.createdAccount == nil {
			return fmt.Errorf("created account outbox handler is not configured")
		}
		return h.createdAccount.Handle(event)
	}
	return fmt.Errorf("unsupported outbox type %q", event.Type)
}
