package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/job"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/ajuda-dev/backend/src/service/notification"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationStream_Unauthorized(t *testing.T) {
	app := setupApp()
	req, _ := http.NewRequest(http.MethodGet, "/v1/notifications/stream", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestNotificationStream_AuthorizedHeaders(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "sse user", Email: "sse_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, createErr)
	app := setupApp()
	req, _ := http.NewRequest(http.MethodGet, "/v1/notifications/stream", nil)
	req.Header.Set("Authorization", "Bearer "+validTokenFor(t, user.Id))
	resp, testErr := app.Test(req, 50)
	if testErr != nil {
		assert.Contains(t, testErr.Error(), "timeout")
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
}

func TestOutboxSSE_PublishAndOfflineSent(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "sse pub", Email: "ssep_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	other, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "sse other", Email: "sseo_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)

	hub := notification.NewNotificationHub()
	chUser, cancelUser := hub.Subscribe(user.Id)
	chOther, cancelOther := hub.Subscribe(other.Id)
	defer cancelUser()
	defer cancelOther()

	payload, _ := json.Marshal(map[string]string{"title": "aviso"})
	event := &notificationdomain.OutboxEventDomain{
		Type: notificationdomain.OutboxTypeCommunityEventPendingApproval, UserId: user.Id, Payload: payload, Status: notificationdomain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))
	job.RunOutboxOnce(outboxEventRepository, job.NewSSEOutboxHandler(hub), 50)

	select {
	case msg := <-chUser:
		assert.Contains(t, string(msg), notificationdomain.OutboxTypeCommunityEventPendingApproval)
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification for target user")
	}
	select {
	case <-chOther:
		t.Fatal("other user must not receive the event")
	case <-time.After(100 * time.Millisecond):
	}

	var stored notificationentity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, notificationdomain.OutboxStatusSent, stored.Status)

	offline := &notificationdomain.OutboxEventDomain{
		Type: notificationdomain.OutboxTypeMentoringInvitePending, UserId: user.Id, Payload: payload, Status: notificationdomain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, offline))
	cancelUser()
	job.RunOutboxOnce(outboxEventRepository, job.NewSSEOutboxHandler(hub), 50)
	var offlineStored notificationentity.OutboxEventEntity
	require.NoError(t, db.First(&offlineStored, offline.Id).Error)
	assert.Equal(t, notificationdomain.OutboxStatusSent, offlineStored.Status)
}
