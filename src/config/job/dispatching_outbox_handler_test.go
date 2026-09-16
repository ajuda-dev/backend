package job

import (
	"errors"
	"testing"

	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routeRecordingHandler struct {
	calls int
	err   error
}

func (h *routeRecordingHandler) Handle(event notificationdomain.OutboxEventDomain) error {
	h.calls++
	return h.err
}

func TestDispatchingOutboxHandler_RoutesInboxToSSE(t *testing.T) {
	inbox := &routeRecordingHandler{}
	created := &routeRecordingHandler{}
	handler := NewDispatchingOutboxHandler(inbox, created)

	require.NoError(t, handler.Handle(notificationdomain.OutboxEventDomain{Type: notificationdomain.OutboxTypeCommunityEventApproved}))
	assert.Equal(t, 1, inbox.calls)
	assert.Equal(t, 0, created.calls)
}

func TestDispatchingOutboxHandler_RoutesCreatedAccount(t *testing.T) {
	inbox := &routeRecordingHandler{}
	created := &routeRecordingHandler{}
	handler := NewDispatchingOutboxHandler(inbox, created)

	require.NoError(t, handler.Handle(notificationdomain.OutboxEventDomain{Type: notificationdomain.OutboxTypeCreatedAccount}))
	assert.Equal(t, 0, inbox.calls)
	assert.Equal(t, 1, created.calls)
}

func TestDispatchingOutboxHandler_UnknownTypeFails(t *testing.T) {
	handler := NewDispatchingOutboxHandler(&routeRecordingHandler{}, &routeRecordingHandler{})
	err := handler.Handle(notificationdomain.OutboxEventDomain{Type: "PASSWORD_RESET"})
	require.Error(t, err)
}

func TestDispatchingOutboxHandler_CreatedAccountHandlerError(t *testing.T) {
	created := &routeRecordingHandler{err: errors.New("smtp down")}
	handler := NewDispatchingOutboxHandler(&routeRecordingHandler{}, created)
	err := handler.Handle(notificationdomain.OutboxEventDomain{Type: notificationdomain.OutboxTypeCreatedAccount})
	require.EqualError(t, err, "smtp down")
}

func TestInboxNotificationTypes_ExcludesCreatedAccount(t *testing.T) {
	assert.False(t, notificationdomain.IsInboxNotificationType(notificationdomain.OutboxTypeCreatedAccount))
	assert.True(t, notificationdomain.IsInboxNotificationType(notificationdomain.OutboxTypeCommunityEventPendingApproval))
}
