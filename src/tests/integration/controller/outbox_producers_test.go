package controller_test

import (
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateEventPending_InsertsOutboxForCommunityOwner(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanCommunityUsersTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "owner", Email: "own_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	member, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "member", Email: "mem_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-outbox")
	community := createEventCommunity(t, "Comunidade outbox "+uuidv7.New().String(), owner, address)
	_, err = communityUserRepository.Create(&domain.CommunityUserDomain{CommunityId: community.Id, UserId: member.Id})
	require.Nil(t, err)

	created, createErr := eventRepository.CreateEvent(&domain.EventDomain{
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "pendente",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *member,
		Community:   community,
		Status:      domain.EventStatusPending,
	})
	require.Nil(t, createErr)
	require.NotEmpty(t, created.Id)

	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	require.Len(t, pending, 1)
	assert.Equal(t, owner.Id, pending[0].UserId)
	assert.Equal(t, domain.OutboxTypeCommunityEventPendingApproval, pending[0].Type)
}

func TestCreateEventApproved_NoOutbox(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventsTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "owner2", Email: "own2_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-approved")
	community := createEventCommunity(t, "Comunidade approved "+uuidv7.New().String(), owner, address)

	_, createErr := eventRepository.CreateEvent(&domain.EventDomain{
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "aprovado",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *owner,
		Community:   community,
		Status:      domain.EventStatusApproved,
	})
	require.Nil(t, createErr)
	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	assert.Empty(t, pending)
}

func TestMentoringInvite_InsertsOutboxForGuest(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "mentor", Email: "men_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	guest, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "guest", Email: "gue_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	slots := 2
	event, err := eventRepository.CreateEvent(&domain.EventDomain{
		Category:    domain.CategoryMentoring,
		Type:        domain.TypeOnline,
		Title:       "1:1",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *mentor,
		MaxSlots:    &slots,
		Status:      domain.EventStatusApproved,
	})
	require.Nil(t, err)

	_, joinErr := eventUserRepository.CreateOrUpdate(&domain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: domain.RoleMentee, Status: domain.StatusRequested,
	}, &slots)
	require.Nil(t, joinErr)

	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	require.Len(t, pending, 1)
	assert.Equal(t, guest.Id, pending[0].UserId)
	assert.Equal(t, domain.OutboxTypeMentoringInvitePending, pending[0].Type)
}

func TestCreateEvent_OutboxFailureRollsBackEvent(t *testing.T) {
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("fail_outbox_payload")
		db.Exec("DELETE FROM outbox_events")
		cleanEventsTable()
		cleanCommunityUsersTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "owner3", Email: "own3_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	member, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "member3", Email: "mem3_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-rb")
	community := createEventCommunity(t, "Comunidade rb "+uuidv7.New().String(), owner, address)
	_, err = communityUserRepository.Create(&domain.CommunityUserDomain{CommunityId: community.Id, UserId: member.Id})
	require.Nil(t, err)

	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_outbox_payload", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*entity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	_, createErr := eventRepository.CreateEvent(&domain.EventDomain{
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "rollback",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *member,
		Community:   community,
		Status:      domain.EventStatusPending,
	})
	require.NotNil(t, createErr)

	var count int64
	require.NoError(t, db.Model(&entity.EventEntity{}).Where("title = ?", "rollback").Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
