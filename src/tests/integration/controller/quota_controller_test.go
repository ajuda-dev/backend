package controller_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/gofiber/fiber/v2"
)

func setQuotaEnv(t *testing.T, pairs map[string]string) {
	t.Helper()
	for key, value := range pairs {
		t.Setenv(key, value)
	}
}

func registerCommunityExpect(t *testing.T, app *fiber.App, token string, addressId string, name string) *http.Response {
	t.Helper()
	body := []byte(`{
	"address_id": "` + addressId + `",
	"name": "` + name + `",
	"description": "comunidade de teste de quota"
	}`)
	resp, err := doAuthedRequest(app, newCommunityRegisterRequest(body), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func registerEventExpect(t *testing.T, app *fiber.App, body eventTestRequest) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, body.OwnerId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func assertTooManyRequests(t *testing.T, resp *http.Response, message string) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("esperava 429, recebeu %d", resp.StatusCode)
	}
	var body rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if body.Message != message {
		t.Fatalf("esperava message '%s', recebeu '%s'", message, body.Message)
	}
}

func TestOwnedCommunitiesQuotaUserModeratorAdmin(t *testing.T) {
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	setQuotaEnv(t, map[string]string{
		"MAX_OWNED_COMMUNITIES":                          "1",
		"MAX_OWNED_COMMUNITIES_MODERATOR":                "2",
		"RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR":           "1000",
		"RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR_MODERATOR": "1000",
	})
	app := setupApp()

	user := createUserWithRole(t, "quota_community_user@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "quota_community_mod@ajuda.dev", userdomain.UserRoleModerator)
	admin := createUserWithRole(t, "quota_community_admin@ajuda.dev", userdomain.UserRoleAdmin)

	userAddr := createCommunityAddress(t, "quota-user-city")
	userToken := validTokenFor(t, user.Id)
	resp := registerCommunityExpect(t, app, userToken, userAddr.Id, "Quota User Community 1")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 na 1a comunidade do USER, recebeu %d", resp.StatusCode)
	}
	assertTooManyRequests(t, registerCommunityExpect(t, app, userToken, createCommunityAddress(t, "quota-user-city-2").Id, "Quota User Community 2"), "owned communities limit reached")

	modToken := validTokenFor(t, moderator.Id)
	resp = registerCommunityExpect(t, app, modToken, createCommunityAddress(t, "quota-mod-city-1").Id, "Quota Mod Community 1")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 na 1a comunidade do MODERATOR, recebeu %d", resp.StatusCode)
	}
	resp = registerCommunityExpect(t, app, modToken, createCommunityAddress(t, "quota-mod-city-2").Id, "Quota Mod Community 2")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 na 2a comunidade do MODERATOR, recebeu %d", resp.StatusCode)
	}
	assertTooManyRequests(t, registerCommunityExpect(t, app, modToken, createCommunityAddress(t, "quota-mod-city-3").Id, "Quota Mod Community 3"), "owned communities limit reached")

	adminToken := validTokenFor(t, admin.Id)
	for i := 1; i <= 3; i++ {
		resp = registerCommunityExpect(t, app, adminToken, createCommunityAddress(t, fmt.Sprintf("quota-admin-city-%d", i)).Id, fmt.Sprintf("Quota Admin Community %d", i))
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("esperava 201 na comunidade %d do ADMIN, recebeu %d", i, resp.StatusCode)
		}
	}
}

func TestCommunityCreateRateLimit(t *testing.T) {
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	setQuotaEnv(t, map[string]string{
		"MAX_OWNED_COMMUNITIES":                          "100",
		"RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR":           "2",
		"RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR_MODERATOR": "100",
	})
	app := setupApp()
	user := createUserWithRole(t, "quota_rate_community@ajuda.dev", userdomain.UserRoleUser)
	token := validTokenFor(t, user.Id)

	resp := registerCommunityExpect(t, app, token, createCommunityAddress(t, "rate-city-1").Id, "Rate Community 1")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d", resp.StatusCode)
	}
	resp = registerCommunityExpect(t, app, token, createCommunityAddress(t, "rate-city-2").Id, "Rate Community 2")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d", resp.StatusCode)
	}
	assertTooManyRequests(t, registerCommunityExpect(t, app, token, createCommunityAddress(t, "rate-city-3").Id, "Rate Community 3"), "too many community creations")
}

func TestPendingAndActiveEventQuotas(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	setQuotaEnv(t, map[string]string{
		"MAX_PENDING_EVENTS":                         "1",
		"MAX_PENDING_EVENTS_MODERATOR":               "2",
		"MAX_ACTIVE_EVENTS":                          "2",
		"MAX_ACTIVE_EVENTS_MODERATOR":                "4",
		"RATE_LIMIT_EVENT_CREATE_PER_HOUR":           "1000",
		"RATE_LIMIT_EVENT_CREATE_PER_HOUR_MODERATOR": "1000",
	})
	app := setupApp()

	owner := createUserWithRole(t, "quota_event_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "quota_event_member@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "quota-event-city")
	community := createEventCommunity(t, "Comunidade quota eventos", owner, address)
	eaAddCommunityMember(t, community.Id, member.Id)

	resp := registerEventExpect(t, app, eventTestRequest{
		OwnerId:     member.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Pending 1",
		Description: "quota",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no 1o PENDING, recebeu %d", resp.StatusCode)
	}
	assertTooManyRequests(t, registerEventExpect(t, app, eventTestRequest{
		OwnerId:     member.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Pending 2",
		Description: "quota",
		StartAt:     time.Now().Add(49 * time.Hour),
		DurationMin: 60,
	}), "pending events limit reached")

	// Owner creates APPROVED events — pending quota does not apply; active does.
	resp = registerEventExpect(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Approved 1",
		Description: "quota",
		StartAt:     time.Now().Add(50 * time.Hour),
		DurationMin: 60,
	})
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no 1o APPROVED do owner, recebeu %d", resp.StatusCode)
	}
	resp = registerEventExpect(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Approved 2",
		Description: "quota",
		StartAt:     time.Now().Add(51 * time.Hour),
		DurationMin: 60,
	})
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no 2o APPROVED do owner, recebeu %d", resp.StatusCode)
	}
	assertTooManyRequests(t, registerEventExpect(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Approved 3",
		Description: "quota",
		StartAt:     time.Now().Add(52 * time.Hour),
		DurationMin: 60,
	}), "active events limit reached")
}

func TestEventCreateRateLimit(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	setQuotaEnv(t, map[string]string{
		"MAX_ACTIVE_EVENTS":                "100",
		"MAX_PENDING_EVENTS":               "100",
		"RATE_LIMIT_EVENT_CREATE_PER_HOUR": "2",
	})
	app := setupApp()
	owner := createUserWithRole(t, "quota_rate_event@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "quota-rate-event-city")
	community := createEventCommunity(t, "Comunidade rate event", owner, address)

	for i := 1; i <= 2; i++ {
		resp := registerEventExpect(t, app, eventTestRequest{
			OwnerId:     owner.Id,
			CommunityId: strPtr(community.Id),
			Category:    eventdomain.CategoryCommunityEvent,
			Type:        eventdomain.TypeOnline,
			Title:       fmt.Sprintf("Rate Event %d", i),
			Description: "quota",
			StartAt:     time.Now().Add(time.Duration(48+i) * time.Hour),
			DurationMin: 60,
		})
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("esperava 201 no evento %d, recebeu %d", i, resp.StatusCode)
		}
	}
	assertTooManyRequests(t, registerEventExpect(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Rate Event 3",
		Description: "quota",
		StartAt:     time.Now().Add(60 * time.Hour),
		DurationMin: 60,
	}), "too many event creations")
}

func TestCommunityMembershipQuotaAndJoinRate(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	setQuotaEnv(t, map[string]string{
		"MAX_COMMUNITY_MEMBERSHIPS":                    "1",
		"MAX_COMMUNITY_MEMBERSHIPS_MODERATOR":          "2",
		"RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR":           "1000",
		"RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR_MODERATOR": "1000",
	})
	app := setupApp()

	owner := createUserWithRole(t, "quota_join_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "quota_join_member@ajuda.dev", userdomain.UserRoleUser)
	c1 := createEventCommunity(t, "Quota Join Community 1", owner, createEventAddress(t, "join-city-1"))
	c2 := createEventCommunity(t, "Quota Join Community 2", owner, createEventAddress(t, "join-city-2"))

	token := validTokenFor(t, member.Id)
	resp, err := doAuthedRequest(app, newJoinCommunityRequest(c1.Id), token)
	if err != nil {
		t.Fatalf("erro ao join: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no 1o join, recebeu %d", resp.StatusCode)
	}
	resp, err = doAuthedRequest(app, newJoinCommunityRequest(c2.Id), token)
	if err != nil {
		t.Fatalf("erro ao join: %v", err)
	}
	assertTooManyRequests(t, resp, "community memberships limit reached")

	setQuotaEnv(t, map[string]string{
		"MAX_COMMUNITY_MEMBERSHIPS":          "100",
		"RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR": "1",
	})
	app = setupApp()
	rateMember := createUserWithRole(t, "quota_join_rate@ajuda.dev", userdomain.UserRoleUser)
	c3 := createEventCommunity(t, "Quota Join Rate Community 1", owner, createEventAddress(t, "join-rate-1"))
	c4 := createEventCommunity(t, "Quota Join Rate Community 2", owner, createEventAddress(t, "join-rate-2"))
	rateToken := validTokenFor(t, rateMember.Id)
	resp, err = doAuthedRequest(app, newJoinCommunityRequest(c3.Id), rateToken)
	if err != nil {
		t.Fatalf("erro ao join: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join com rate, recebeu %d", resp.StatusCode)
	}
	resp, err = doAuthedRequest(app, newJoinCommunityRequest(c4.Id), rateToken)
	if err != nil {
		t.Fatalf("erro ao join: %v", err)
	}
	assertTooManyRequests(t, resp, "too many community joins")
}

func TestSkillsOptionalQuota(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	t.Setenv("MAX_SKILLS_PER_USER", "")
	t.Setenv("MAX_SKILLS_PER_USER_MODERATOR", "")
	app := setupApp()
	user := createUserWithRole(t, "quota_skill_unlimited@ajuda.dev", userdomain.UserRoleUser)
	for i := 1; i <= 3; i++ {
		skill := createSkillForTest(t, fmt.Sprintf("QUOTAUNLIM%d", i))
		assignSkillViaApi(t, app, skill.Id, user.Id, skilldomain.LevelTeach)
	}

	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Setenv("MAX_SKILLS_PER_USER", "2")
	t.Setenv("MAX_SKILLS_PER_USER_MODERATOR", "3")
	app = setupApp()

	limited := createUserWithRole(t, "quota_skill_user@ajuda.dev", userdomain.UserRoleUser)
	s1 := createSkillForTest(t, "QUOTALIM1")
	s2 := createSkillForTest(t, "QUOTALIM2")
	s3 := createSkillForTest(t, "QUOTALIM3")
	assignSkillViaApi(t, app, s1.Id, limited.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, s2.Id, limited.Id, skilldomain.LevelTeach)
	resp := doAssignSkill(t, app, s3.Id, assignSkillBody(t, limited.Id, skilldomain.LevelTeach), validTokenFor(t, limited.Id))
	assertTooManyRequests(t, resp, "skills limit reached")

	moderator := createUserWithRole(t, "quota_skill_mod@ajuda.dev", userdomain.UserRoleModerator)
	ms1 := createSkillForTest(t, "QUOTAMOD1")
	ms2 := createSkillForTest(t, "QUOTAMOD2")
	ms3 := createSkillForTest(t, "QUOTAMOD3")
	ms4 := createSkillForTest(t, "QUOTAMOD4")
	assignSkillViaApi(t, app, ms1.Id, moderator.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, ms2.Id, moderator.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, ms3.Id, moderator.Id, skilldomain.LevelTeach)
	resp = doAssignSkill(t, app, ms4.Id, assignSkillBody(t, moderator.Id, skilldomain.LevelTeach), validTokenFor(t, moderator.Id))
	assertTooManyRequests(t, resp, "skills limit reached")

	admin := createUserWithRole(t, "quota_skill_admin@ajuda.dev", userdomain.UserRoleAdmin)
	for i := 1; i <= 4; i++ {
		skill := createSkillForTest(t, fmt.Sprintf("QUOTAADMIN%d", i))
		assignSkillViaApi(t, app, skill.Id, admin.Id, skilldomain.LevelTeach)
	}
}
