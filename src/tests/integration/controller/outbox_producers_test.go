package controller_test

import (
	"encoding/json"
	"testing"
	"time"

	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
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
	owner, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "owner", Email: "own_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	member, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "member", Email: "mem_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-outbox")
	community := createEventCommunity(t, "Comunidade outbox "+uuidv7.New().String(), owner, address)
	_, err = communityUserRepository.Create(&communitydomain.CommunityUserDomain{CommunityId: community.Id, UserId: member.Id})
	require.Nil(t, err)

	created, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "pendente",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *member,
		Community:   community,
		Status:      eventdomain.EventStatusPending,
	})
	require.Nil(t, createErr)
	require.NotEmpty(t, created.Id)

	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	require.Len(t, pending, 1)
	assert.Equal(t, owner.Id, pending[0].UserId)
	assert.Equal(t, notificationdomain.OutboxTypeCommunityEventPendingApproval, pending[0].Type)
}

func TestCreateEventApproved_NoOutbox(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventsTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "owner2", Email: "own2_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-approved")
	community := createEventCommunity(t, "Comunidade approved "+uuidv7.New().String(), owner, address)

	_, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "aprovado",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *owner,
		Community:   community,
		Status:      eventdomain.EventStatusApproved,
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
	mentor, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "mentor", Email: "men_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	guest, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "guest", Email: "gue_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	slots := 2
	event, err := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "1:1",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *mentor,
		MaxSlots:    &slots,
		Status:      eventdomain.EventStatusApproved,
	})
	require.Nil(t, err)

	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, &slots)
	require.Nil(t, joinErr)

	pending, findErr := outboxEventRepository.FindPending(10)
	require.Nil(t, findErr)
	require.Len(t, pending, 1)
	assert.Equal(t, guest.Id, pending[0].UserId)
	assert.Equal(t, notificationdomain.OutboxTypeMentoringInvitePending, pending[0].Type)
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
	owner, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "owner3", Email: "own3_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	member, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "member3", Email: "mem3_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-rb")
	community := createEventCommunity(t, "Comunidade rb "+uuidv7.New().String(), owner, address)
	_, err = communityUserRepository.Create(&communitydomain.CommunityUserDomain{CommunityId: community.Id, UserId: member.Id})
	require.Nil(t, err)

	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_outbox_payload", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*notificationentity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	_, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "rollback",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *member,
		Community:   community,
		Status:      eventdomain.EventStatusPending,
	})
	require.NotNil(t, createErr)

	var count int64
	require.NoError(t, db.Model(&evententity.EventEntity{}).Where("title = ?", "rollback").Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestMentoringInvite_Accept_NotifiesOtherParticipants(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, guest, event, slots := setupMentoringInvite(t)

	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)

	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusConfirmed, "", slots)
	require.Nil(t, updateErr)

	accepted := pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInviteAccepted)
	require.Len(t, accepted, 1)
	assertOutboxPayload(t, accepted[0].Payload, map[string]string{
		"event_id": event.Id,
		"title":    event.Title,
		"category": eventdomain.CategoryMentoring,
		"status":   eventdomain.StatusConfirmed,
		"actor_id": guest.Id,
	})
	assert.NotContains(t, string(accepted[0].Payload), `"comment"`)
	assert.Empty(t, pendingOutboxByUserAndType(t, guest.Id, notificationdomain.OutboxTypeMentoringInviteAccepted))
	assert.Len(t, pendingOutboxByUserAndType(t, guest.Id, notificationdomain.OutboxTypeMentoringInvitePending), 1)
}

func TestMentoringInvite_Reject_NotifiesOtherParticipants(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, guest, event, slots := setupMentoringInvite(t)

	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)

	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusRejected, "Agenda conflitou nesta semana", slots)
	require.Nil(t, updateErr)

	rejected := pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInviteRejected)
	require.Len(t, rejected, 1)
	assertOutboxPayload(t, rejected[0].Payload, map[string]string{
		"event_id": event.Id,
		"title":    event.Title,
		"category": eventdomain.CategoryMentoring,
		"status":   eventdomain.StatusRejected,
		"actor_id": guest.Id,
	})
	assert.NotContains(t, string(rejected[0].Payload), `"comment"`)
	assert.Empty(t, pendingOutboxByUserAndType(t, guest.Id, notificationdomain.OutboxTypeMentoringInviteRejected))
}

func TestMentoringInvite_Cancel_NoResponseOutbox(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, guest, event, slots := setupMentoringInvite(t)

	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)

	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusCancelled, "", slots)
	require.Nil(t, updateErr)

	assert.Empty(t, pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInviteAccepted))
	assert.Empty(t, pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInviteRejected))
	assert.Len(t, pendingOutboxByUserAndType(t, guest.Id, notificationdomain.OutboxTypeMentoringInvitePending), 1)
}

func TestCommunityEvent_Approve_NotifiesEventOwner(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanCommunityUsersTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, member, created := setupPendingCommunityEvent(t)

	updated, updateErr := eventRepository.UpdateApprovalStatus(created.Id, eventdomain.EventStatusPending, eventdomain.EventStatusApproved, owner.Id)
	require.Nil(t, updateErr)
	assert.Equal(t, eventdomain.EventStatusApproved, updated.Status)

	approved := pendingOutboxByUserAndType(t, member.Id, notificationdomain.OutboxTypeCommunityEventApproved)
	require.Len(t, approved, 1)
	assertOutboxPayload(t, approved[0].Payload, map[string]string{
		"event_id":     created.Id,
		"title":        created.Title,
		"community_id": created.Community.Id,
		"category":     created.Category,
		"status":       eventdomain.EventStatusApproved,
		"actor_id":     owner.Id,
	})
	assert.Empty(t, pendingOutboxByUserAndType(t, owner.Id, notificationdomain.OutboxTypeCommunityEventApproved))
	assert.Len(t, pendingOutboxByUserAndType(t, owner.Id, notificationdomain.OutboxTypeCommunityEventPendingApproval), 1)
}

func TestCommunityEvent_Reject_NotifiesEventOwner(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanCommunityUsersTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, member, created := setupPendingCommunityEvent(t)

	updated, updateErr := eventRepository.UpdateApprovalStatus(created.Id, eventdomain.EventStatusPending, eventdomain.EventStatusRejected, owner.Id)
	require.Nil(t, updateErr)
	assert.Equal(t, eventdomain.EventStatusRejected, updated.Status)

	rejected := pendingOutboxByUserAndType(t, member.Id, notificationdomain.OutboxTypeCommunityEventRejected)
	require.Len(t, rejected, 1)
	assert.Equal(t, eventdomain.EventStatusRejected, payloadString(t, rejected[0].Payload, "status"))
	assert.Empty(t, pendingOutboxByUserAndType(t, owner.Id, notificationdomain.OutboxTypeCommunityEventRejected))
}

func TestUpdateApproval_OutboxFailureRollsBackStatus(t *testing.T) {
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("fail_approval_outbox_payload")
		db.Exec("DELETE FROM outbox_events")
		cleanEventsTable()
		cleanCommunityUsersTable()
		cleanCommunityTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	owner, _, created := setupPendingCommunityEvent(t)

	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_approval_outbox_payload", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*notificationentity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	_, updateErr := eventRepository.UpdateApprovalStatus(created.Id, eventdomain.EventStatusPending, eventdomain.EventStatusApproved, owner.Id)
	require.NotNil(t, updateErr)

	stored, findErr := eventRepository.FindById(created.Id)
	require.Nil(t, findErr)
	assert.Equal(t, eventdomain.EventStatusPending, stored.Status)
	assert.Empty(t, pendingOutboxByUserAndType(t, created.Owner.Id, notificationdomain.OutboxTypeCommunityEventApproved))
}

func TestMentoringInvite_Accept_OutboxFailureRollsBackStatus(t *testing.T) {
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("fail_invite_accept_outbox_payload")
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	_, guest, event, slots := setupMentoringInvite(t)
	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)

	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_invite_accept_outbox_payload", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*notificationentity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusConfirmed, "", slots)
	require.NotNil(t, updateErr)

	participants, findErr := eventUserRepository.FindByEvent(event.Id, "")
	require.Nil(t, findErr)
	var guestStatus string
	for _, participant := range participants {
		if participant.UserId == guest.Id {
			guestStatus = participant.Status
		}
	}
	assert.Equal(t, eventdomain.StatusRequested, guestStatus)
}

func TestMentoringReschedule_InsertsRescheduledForCounterpart(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, guest, event, slots := setupMentoringInvite(t)
	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)
	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusConfirmed, "aceito", slots)
	require.Nil(t, updateErr)

	db.Exec("DELETE FROM outbox_events")
	newStart := time.Now().Add(72 * time.Hour)
	_, rescheduleErr := eventRepository.Reschedule(event.Id, newStart, "Sexta 15h encaixa melhor", guest.Id)
	require.Nil(t, rescheduleErr)

	pending := pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInviteRescheduled)
	require.Len(t, pending, 1)
	assertOutboxPayload(t, pending[0].Payload, map[string]string{
		"event_id": event.Id,
		"title":    event.Title,
		"category": eventdomain.CategoryMentoring,
	})
	assert.Empty(t, pendingOutboxByUserAndType(t, guest.Id, notificationdomain.OutboxTypeMentoringInviteRescheduled))
	assert.Empty(t, pendingOutboxByUserAndType(t, mentor.Id, notificationdomain.OutboxTypeMentoringInvitePending))

	participants, findErr := eventUserRepository.FindByEvent(event.Id, "")
	require.Nil(t, findErr)
	statusByUser := map[string]string{}
	for _, participant := range participants {
		statusByUser[participant.UserId] = participant.Status
	}
	assert.Equal(t, eventdomain.StatusRequested, statusByUser[mentor.Id])
	assert.Equal(t, eventdomain.StatusConfirmed, statusByUser[guest.Id])
}

func TestMentoringReschedule_OutboxFailureRollsBackEventAndStatuses(t *testing.T) {
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("fail_reschedule_outbox_payload")
		db.Exec("DELETE FROM outbox_events")
		cleanEventUsersTable()
		cleanEventsTable()
		cleanAddressesTable()
		cleanUsersTable()
	})
	mentor, guest, event, slots := setupMentoringInvite(t)
	_, joinErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: guest.Id, Role: eventdomain.RoleMentee, Status: eventdomain.StatusRequested,
	}, slots)
	require.Nil(t, joinErr)
	_, updateErr := eventUserRepository.UpdateStatus(event.Id, guest.Id, eventdomain.StatusConfirmed, "", slots)
	require.Nil(t, updateErr)

	before, findBeforeErr := eventRepository.FindById(event.Id)
	require.Nil(t, findBeforeErr)
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_reschedule_outbox_payload", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*notificationentity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	_, rescheduleErr := eventRepository.Reschedule(event.Id, time.Now().Add(72*time.Hour), "Sexta 15h encaixa melhor", guest.Id)
	require.NotNil(t, rescheduleErr)

	stored, findErr := eventRepository.FindById(event.Id)
	require.Nil(t, findErr)
	assert.WithinDuration(t, before.StartAt, stored.StartAt, time.Second)
	assert.Equal(t, before.Comment, stored.Comment)

	participants, partErr := eventUserRepository.FindByEvent(event.Id, "")
	require.Nil(t, partErr)
	statusByUser := map[string]string{}
	for _, participant := range participants {
		statusByUser[participant.UserId] = participant.Status
	}
	assert.Equal(t, eventdomain.StatusConfirmed, statusByUser[mentor.Id])
	assert.Equal(t, eventdomain.StatusConfirmed, statusByUser[guest.Id])
}

func setupMentoringInvite(t *testing.T) (*userdomain.UserDomain, *userdomain.UserDomain, *eventdomain.EventDomain, *int) {
	t.Helper()
	mentor, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "mentor", Email: "men_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	guest, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "guest", Email: "gue_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	slots := 2
	event, err := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "1:1",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *mentor,
		MaxSlots:    &slots,
		Status:      eventdomain.EventStatusApproved,
	})
	require.Nil(t, err)
	_, ownerErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: event.Id, UserId: mentor.Id, Role: eventdomain.RoleMentor, Status: eventdomain.StatusConfirmed,
	}, &slots)
	require.Nil(t, ownerErr)
	return mentor, guest, event, &slots
}

func setupPendingCommunityEvent(t *testing.T) (*userdomain.UserDomain, *userdomain.UserDomain, *eventdomain.EventDomain) {
	t.Helper()
	owner, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "owner", Email: "own_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	member, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name: "member", Email: "mem_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	address := createEventAddress(t, "cidade-approval-outbox")
	community := createEventCommunity(t, "Comunidade approval "+uuidv7.New().String(), owner, address)
	_, err = communityUserRepository.Create(&communitydomain.CommunityUserDomain{CommunityId: community.Id, UserId: member.Id})
	require.Nil(t, err)

	created, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "pendente",
		Description: "d",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		Owner:       *member,
		Community:   community,
		Status:      eventdomain.EventStatusPending,
	})
	require.Nil(t, createErr)
	return owner, member, created
}

func pendingOutboxByUserAndType(t *testing.T, userId, typ string) []notificationdomain.OutboxEventDomain {
	t.Helper()
	pending, err := outboxEventRepository.FindPending(50)
	require.Nil(t, err)
	matched := make([]notificationdomain.OutboxEventDomain, 0)
	for _, event := range pending {
		if event.UserId == userId && event.Type == typ {
			matched = append(matched, event)
		}
	}
	return matched
}

func assertOutboxPayload(t *testing.T, raw json.RawMessage, expected map[string]string) {
	t.Helper()
	var payload map[string]string
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, expected, payload)
}

func payloadString(t *testing.T, raw json.RawMessage, key string) string {
	t.Helper()
	var payload map[string]string
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload[key]
}
