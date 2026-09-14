package controller_test

import (
	"encoding/json"
	"testing"

	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxEventRepository_CreateAndFindPending(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})

	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name:     "outbox user",
		Email:    "outbox_" + uuidv7.New().String() + "@ajuda.dev",
		Password: "123456",
	})
	require.Nil(t, err)

	payload, _ := json.Marshal(map[string]string{"title": "evento"})
	event := &domain.OutboxEventDomain{
		Type:    domain.OutboxTypeCommunityEventPendingApproval,
		UserId:  user.Id,
		Payload: payload,
		Status:  domain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))
	assert.NotZero(t, event.Id)

	var colType string
	require.NoError(t, db.Raw(
		"SELECT data_type FROM information_schema.columns WHERE table_name = 'outbox_events' AND column_name = 'payload'",
	).Scan(&colType).Error)
	assert.Equal(t, "jsonb", colType)

	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	require.Len(t, pending, 1)
	assert.Equal(t, domain.OutboxStatusPending, pending[0].Status)
	assert.Equal(t, user.Id, pending[0].UserId)
	assert.Equal(t, domain.OutboxTypeCommunityEventPendingApproval, pending[0].Type)

	require.Nil(t, outboxEventRepository.UpdateStatus(event.Id, domain.OutboxStatusPending, domain.OutboxStatusProcessing))
	pending, findErr = outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	assert.Empty(t, pending)

	var stored entity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, domain.OutboxStatusProcessing, stored.Status)
}
