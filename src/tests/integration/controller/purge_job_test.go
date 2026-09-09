package controller_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
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

func createPurgeUser(t *testing.T) *domain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
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
	user      *domain.UserDomain
	address   *domain.AddressDomain
	community *domain.CommunityDomain
}

func newPurgeFixture(t *testing.T, label string) purgeFixture {
	t.Helper()
	user := createPurgeUser(t)
	address := createEventAddress(t, "cidade purge "+label)
	community := createEventCommunity(t, "Comunidade purge "+label, user, address)
	return purgeFixture{user: user, address: address, community: community}
}

func createPurgeEvent(t *testing.T, owner *domain.UserDomain, community *domain.CommunityDomain, title string) *domain.EventDomain {
	t.Helper()
	event, createErr := eventRepository.CreateEvent(&domain.EventDomain{
		Owner:       *owner,
		Community:   &domain.CommunityDomain{Id: community.Id},
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
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
	participation, createErr := eventUserRepository.CreateOrUpdate(&domain.EventUserDomain{
		EventId: eventId,
		UserId:  userId,
		Role:    domain.RoleAttendee,
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
	participationId := createPurgeParticipation(t, event.Id, fx.user.Id, domain.StatusConfirmed)

	backdateDeletedAt(t, &entity.UserEntity{}, fx.user.Id, 40)
	backdateDeletedAt(t, &entity.AddressEntity{}, fx.address.Id, 40)
	backdateDeletedAt(t, &entity.CommunityEntity{}, fx.community.Id, 40)
	backdateDeletedAt(t, &entity.EventEntity{}, event.Id, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &entity.UserEntity{}, fx.user.Id); got != 0 {
		t.Errorf("esperava user fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.AddressEntity{}, fx.address.Id); got != 0 {
		t.Errorf("esperava address fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.CommunityEntity{}, fx.community.Id); got != 0 {
		t.Errorf("esperava community fisicamente removida, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventEntity{}, event.Id); got != 0 {
		t.Errorf("esperava event fisicamente removido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventUserEntity{}, participationId); got != 0 {
		t.Errorf("esperava participação removida via cascade, count=%d", got)
	}
}

func TestPurgeKeepsCommunityWithActiveEvent(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "comunidade-protegida")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento ativo da comunidade")
	backdateDeletedAt(t, &entity.CommunityEntity{}, fx.community.Id, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &entity.CommunityEntity{}, fx.community.Id); got != 1 {
		t.Errorf("esperava community arquivada mantida com evento ativo, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava evento ativo mantido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.UserEntity{}, fx.user.Id); got != 1 {
		t.Errorf("esperava user mantido (referenciado), count=%d", got)
	}
	if got := countUnscoped(t, &entity.AddressEntity{}, fx.address.Id); got != 1 {
		t.Errorf("esperava address mantido (referenciado), count=%d", got)
	}
}

func TestPurgeKeepsRecentRecords(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "recentes")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento arquivado recente")
	participationId := createPurgeParticipation(t, event.Id, fx.user.Id, domain.StatusCancelled)

	backdateDeletedAt(t, &entity.UserEntity{}, fx.user.Id, 1)
	backdateDeletedAt(t, &entity.AddressEntity{}, fx.address.Id, 1)
	backdateDeletedAt(t, &entity.CommunityEntity{}, fx.community.Id, 1)
	backdateDeletedAt(t, &entity.EventEntity{}, event.Id, 1)
	backdateUpdatedAt(t, &entity.EventUserEntity{}, participationId, 1)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &entity.UserEntity{}, fx.user.Id); got != 1 {
		t.Errorf("esperava user recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.AddressEntity{}, fx.address.Id); got != 1 {
		t.Errorf("esperava address recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.CommunityEntity{}, fx.community.Id); got != 1 {
		t.Errorf("esperava community recente mantida, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava event recente mantido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventUserEntity{}, participationId); got != 1 {
		t.Errorf("esperava participação recente mantida, count=%d", got)
	}
}

func TestStartPurgeJobDisabled(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "disabled")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento arquivado antigo")
	backdateDeletedAt(t, &entity.UserEntity{}, fx.user.Id, 40)
	backdateDeletedAt(t, &entity.AddressEntity{}, fx.address.Id, 40)
	backdateDeletedAt(t, &entity.CommunityEntity{}, fx.community.Id, 40)
	backdateDeletedAt(t, &entity.EventEntity{}, event.Id, 40)

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()
	job.StartPurgeJob(db, job.PurgeConfig{Enabled: false, OlderThanDays: 30, IntervalHours: 24})
	time.Sleep(200 * time.Millisecond)
	after := runtime.NumGoroutine()

	if after > before {
		t.Errorf("StartPurgeJob com Enabled=false não deveria iniciar goroutine (before=%d, after=%d)", before, after)
	}
	if got := countUnscoped(t, &entity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava nenhuma purga com job desabilitado, count=%d", got)
	}
}

func TestPurgeRemovesOldCancelledParticipations(t *testing.T) {
	purgeTestCleanups(t)
	fx := newPurgeFixture(t, "participacoes")
	event := createPurgeEvent(t, fx.user, fx.community, "Evento vivo com inscrições velhas")
	otherUser := createPurgeUser(t)
	thirdUser := createPurgeUser(t)
	cancelled := createPurgeParticipation(t, event.Id, otherUser.Id, domain.StatusCancelled)
	rejected := createPurgeParticipation(t, event.Id, thirdUser.Id, domain.StatusRejected)
	backdateUpdatedAt(t, &entity.EventUserEntity{}, cancelled, 40)
	backdateUpdatedAt(t, &entity.EventUserEntity{}, rejected, 40)

	job.RunPurge(db, 30)

	if got := countUnscoped(t, &entity.EventUserEntity{}, cancelled); got != 0 {
		t.Errorf("esperava inscrição CANCELLED antiga removida, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventUserEntity{}, rejected); got != 0 {
		t.Errorf("esperava inscrição REJECTED antiga removida, count=%d", got)
	}
	if got := countUnscoped(t, &entity.EventEntity{}, event.Id); got != 1 {
		t.Errorf("esperava evento vivo mantido, count=%d", got)
	}
	if got := countUnscoped(t, &entity.UserEntity{}, otherUser.Id); got != 1 {
		t.Errorf("esperava user de inscrição mantido, count=%d", got)
	}
}
