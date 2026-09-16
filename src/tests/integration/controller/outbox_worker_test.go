package controller_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ajuda-dev/backend/src/config/job"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingHandler struct {
	err error
}

func (h recordingHandler) Handle(event notificationdomain.OutboxEventDomain) error {
	return h.err
}

func TestRunOutboxOnce_MarksSent(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "outbox worker", Email: "ow_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	payload, _ := json.Marshal(map[string]string{"title": "x"})
	event := &notificationdomain.OutboxEventDomain{
		Type: notificationdomain.OutboxTypeCommunityEventPendingApproval, UserId: user.Id, Payload: payload, Status: notificationdomain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))

	job.RunOutboxOnce(outboxEventRepository, recordingHandler{}, 50)

	var stored notificationentity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, notificationdomain.OutboxStatusSent, stored.Status)
}

func TestRunOutboxOnce_HandlerErrorMarksFailed(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "outbox fail", Email: "of_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	payload, _ := json.Marshal(map[string]string{"title": "x"})
	event := &notificationdomain.OutboxEventDomain{
		Type: notificationdomain.OutboxTypeCommunityEventPendingApproval, UserId: user.Id, Payload: payload, Status: notificationdomain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))

	job.RunOutboxOnce(outboxEventRepository, recordingHandler{err: errors.New("boom")}, 50)

	var stored notificationentity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, notificationdomain.OutboxStatusFailed, stored.Status)
}
