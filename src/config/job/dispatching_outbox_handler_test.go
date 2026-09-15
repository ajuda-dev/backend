package job

import (
	"errors"
	"testing"

	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routeRecordingHandler struct {
	calls int
	err   error
}

func (h *routeRecordingHandler) Handle(event domain.OutboxEventDomain) error {
	h.calls++
	return h.err
}

func TestDispatchingOutboxHandler_RoutesInboxToSSE(t *testing.T) {
	inbox := &routeRecordingHandler{}
	created := &routeRecordingHandler{}
	handler := NewDispatchingOutboxHandler(inbox, created)

	require.NoError(t, handler.Handle(domain.OutboxEventDomain{Type: domain.OutboxTypeCommunityEventApproved}))
	assert.Equal(t, 1, inbox.calls)
	assert.Equal(t, 0, created.calls)
}

func TestDispatchingOutboxHandler_RoutesCreatedAccount(t *testing.T) {
	inbox := &routeRecordingHandler{}
	created := &routeRecordingHandler{}
	handler := NewDispatchingOutboxHandler(inbox, created)

	require.NoError(t, handler.Handle(domain.OutboxEventDomain{Type: domain.OutboxTypeCreatedAccount}))
	assert.Equal(t, 0, inbox.calls)
	assert.Equal(t, 1, created.calls)
}

func TestDispatchingOutboxHandler_UnknownTypeFails(t *testing.T) {
	handler := NewDispatchingOutboxHandler(&routeRecordingHandler{}, &routeRecordingHandler{})
	err := handler.Handle(domain.OutboxEventDomain{Type: "PASSWORD_RESET"})
	require.Error(t, err)
}

func TestDispatchingOutboxHandler_CreatedAccountHandlerError(t *testing.T) {
	created := &routeRecordingHandler{err: errors.New("smtp down")}
	handler := NewDispatchingOutboxHandler(&routeRecordingHandler{}, created)
	err := handler.Handle(domain.OutboxEventDomain{Type: domain.OutboxTypeCreatedAccount})
	require.EqualError(t, err, "smtp down")
}

func TestInboxNotificationTypes_ExcludesCreatedAccount(t *testing.T) {
	assert.False(t, domain.IsInboxNotificationType(domain.OutboxTypeCreatedAccount))
	assert.True(t, domain.IsInboxNotificationType(domain.OutboxTypeCommunityEventPendingApproval))
}
