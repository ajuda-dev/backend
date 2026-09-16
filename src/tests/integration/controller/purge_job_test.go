package controller_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/job"
	addressentity "github.com/ajuda-dev/backend/src/data/address/entity"
	communityentity "github.com/ajuda-dev/backend/src/data/community/entity"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/samborkent/uuidv7"
)

func purgeTestCleanups(t *testing.T) {
	t.Helper()
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)
	t.Cleanup(cleanEventUsersTable)
}

func createPurgeUser(t *testing.T) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "usuario purge",
		Email:    "purge_" + strings.ReplaceAll(uuidv7.New().String(), "-", "") + "@ajuda.dev",
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

type purgeFixture struct {
	user      *userdomain.UserDomain
	address   *addressdomain.AddressDomain
	community *communitydomain.CommunityDomain
}

func newPurgeFixture(t *testing.T, label string) purgeFixture {
	t.Helper()
	user := createPurgeUser(t)
	address := createEventAddress(t, "cidade purge "+label)
	community := createEventCommunity(t, "Comunidade purge "+label, user, address)
	return purgeFixture{user: user, address: address, community: community}
}

func createPurgeEvent(t *testing.T, owner *userdomain.UserDomain, community *communitydomain.CommunityDomain, title string) *eventdomain.EventDomain {
	t.Helper()
	event, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Owner:       *owner,
		Community:   &communitydomain.CommunityDomain{Id: community.Id},
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       title,
		Description: "evento para teste de purge",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if createErr != nil {
		t.Fatalf("failed to create event: %v", createErr)
	}
	return event
}

func createPurgeParticipation(t *testing.T, eventId string, userId string, status string) string {
	t.Helper()
	participation, createErr := eventUserRepository.CreateOrUpdate(&eventdomain.EventUserDomain{
		EventId: eventId,
		UserId:  userId,
		Role:    eventdomain.RoleAttendee,
		Status:  status,
	}, nil)
	if createErr != nil {
		t.Fatalf("failed to create participation: %v", createErr)
	}
	return participation.Id
}

func backdateDeletedAt(t *testing.T, model interface{}, id string, days int) {
	t.Helper()
	if err := db.Unscoped().Model(model).Where("id = ?", id).
		Update("deleted_at", time.Now().AddDate(0, 0, -days)).Error; err != nil {
		t.Fatalf("failed to backdate deleted_at: %v", err)
	}
}

func backdateUpdatedAt(t *testing.T, model interface{}, id string, days int) {
	t.Helper()
	if err := db.Model(model).Where("id = ?", id).
		Update("updated_at", time.Now().AddDate(0, 0, -days)).Error; err != nil {
		t.Fatalf("failed to backdate updated_at: %v", err)
	}
}

func countUnscoped(t *testing.T, model interface{}, id string) int64 {
	t.Helper()
	var count int64
	if err := db.Unscoped().Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	return count
}

func TestPurgeRemovesOldArchivedRows(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "antigos")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento antigo arquivado")
	participationId := createPurgeParticipation(t, event.Id, fx.user.Id, eventdomain.StatusConfirmed)

	backdateDeletedAt(t, &userentity.UserEntity{}, fx.user.Id, 40)
	backdateDeletedAt(t, &addressentity.AddressEntity{}, fx.address.Id, 40)
	backdateDeletedAt(t, &communityentity.CommunityEntity{}, fx.community.Id, 40)
	backdateDeletedAt(t, &evententity.EventEntity{}, event.Id, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &userentity.UserEntity{}, fx.user.Id); got != 0 {
		t.Errorf("esperava user fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &addressentity.AddressEntity{}, fx.address.Id); got != 0 {
		t.Errorf("esperava address fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &communityentity.CommunityEntity{}, fx.community.Id); got != 0 {
		t.Errorf("esperava community fisicamente removida, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventEntity{}, event.Id); got != 0 {
		t.Errorf("esperava event fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventUserEntity{}, participationId); got != 0 {
		t.Errorf("esperava participação removida via cascade, count=%d", got)
	}
}

func TestPurgeKeepsCommunityWithActiveEvent(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "comunidade-protegida")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento ativo da comunidade")
	backdateDeletedAt(t, &communityentity.CommunityEntity{}, fx.community.Id, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &communityentity.CommunityEntity{}, fx.community.Id); got != 1 {
		t.Errorf("esperava community arquivada mantida com evento ativo, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava evento ativo mantido, count=%d", got)
	}
	if got := countUnscoped(t, &userentity.UserEntity{}, fx.user.Id); got != 1 {
		t.Errorf("esperava user mantido (referenciado), count=%d", got)
	}
	if got := countUnscoped(t, &addressentity.AddressEntity{}, fx.address.Id); got != 1 {
		t.Errorf("esperava address mantido (referenciado), count=%d", got)
	}
}

func TestPurgeKeepsRecentRecords(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "recentes")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento arquivado recente")
	participationId := createPurgeParticipation(t, event.Id, fx.user.Id, eventdomain.StatusCancelled)

	backdateDeletedAt(t, &userentity.UserEntity{}, fx.user.Id, 1)
	backdateDeletedAt(t, &addressentity.AddressEntity{}, fx.address.Id, 1)
	backdateDeletedAt(t, &communityentity.CommunityEntity{}, fx.community.Id, 1)
	backdateDeletedAt(t, &evententity.EventEntity{}, event.Id, 1)
	backdateUpdatedAt(t, &evententity.EventUserEntity{}, participationId, 1)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &userentity.UserEntity{}, fx.user.Id); got != 1 {
		t.Errorf("esperava user recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &addressentity.AddressEntity{}, fx.address.Id); got != 1 {
		t.Errorf("esperava address recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &communityentity.CommunityEntity{}, fx.community.Id); got != 1 {
		t.Errorf("esperava community recente mantida, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava event recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventUserEntity{}, participationId); got != 1 {
		t.Errorf("esperava participação recente mantida, count=%d", got)
	}
}

func TestStartPurgeJobDisabled(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "disabled")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento arquivado antigo")
	backdateDeletedAt(t, &userentity.UserEntity{}, fx.user.Id, 40)
	backdateDeletedAt(t, &addressentity.AddressEntity{}, fx.address.Id, 40)
	backdateDeletedAt(t, &communityentity.CommunityEntity{}, fx.community.Id, 40)
	backdateDeletedAt(t, &evententity.EventEntity{}, event.Id, 40)

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()
	job.StartPurgeJob(db, job.PurgeConfig{Enabled: false, OlderThanDays: 30, IntervalHours: 24})
	time.Sleep(200 * time.Millisecond)
	after := runtime.NumGoroutine()

	if after > before {
		t.Errorf("StartPurgeJob com Enabled=false não deveria iniciar goroutine (before=%d, after=%d)", before, after)
	}
	if got := countUnscoped(t, &evententity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava nenhuma purga com job desabilitado, count=%d", got)
	}
}

func TestPurgeRemovesOldCancelledParticipations(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "participacoes")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento vivo com inscrições velhas")
	otherUser := createPurgeUser(t)
	thirdUser := createPurgeUser(t)
	cancelled := createPurgeParticipation(t, event.Id, otherUser.Id, eventdomain.StatusCancelled)
	rejected := createPurgeParticipation(t, event.Id, thirdUser.Id, eventdomain.StatusRejected)
	backdateUpdatedAt(t, &evententity.EventUserEntity{}, cancelled, 40)
	backdateUpdatedAt(t, &evententity.EventUserEntity{}, rejected, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &evententity.EventUserEntity{}, cancelled); got != 0 {
		t.Errorf("esperava inscrição CANCELLED antiga removida, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventUserEntity{}, rejected); got != 0 {
		t.Errorf("esperava inscrição REJECTED antiga removida, count=%d", got)
	}
	if got := countUnscoped(t, &evententity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava evento vivo mantido, count=%d", got)
	}
	if got := countUnscoped(t, &userentity.UserEntity{}, otherUser.Id); got != 1 {
		t.Errorf("esperava user de inscrição mantido, count=%d", got)
	}
}
