package controller_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingHandler struct {
	err error
}

func (h recordingHandler) Handle(event domain.OutboxEventDomain) error {
	return h.err
}

func TestRunOutboxOnce_MarksSent(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "outbox worker", Email: "ow_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	payload, _ := json.Marshal(map[string]string{"title": "x"})
	event := &domain.OutboxEventDomain{
		Type: domain.OutboxTypeCommunityEventPendingApproval, UserId: user.Id, Payload: payload, Status: domain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))

	job.RunOutboxOnce(outboxEventRepository, recordingHandler{}, 50)

	var stored entity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, domain.OutboxStatusSent, stored.Status)
}

func TestRunOutboxOnce_HandlerErrorMarksFailed(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "outbox fail", Email: "of_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	payload, _ := json.Marshal(map[string]string{"title": "x"})
	event := &domain.OutboxEventDomain{
		Type: domain.OutboxTypeCommunityEventPendingApproval, UserId: user.Id, Payload: payload, Status: domain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))

	job.RunOutboxOnce(outboxEventRepository, recordingHandler{err: errors.New("boom")}, 50)

	var stored entity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, domain.OutboxStatusFailed, stored.Status)
}
