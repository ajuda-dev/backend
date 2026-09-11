package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
	"golang.org/x/crypto/bcrypt"
)

func cleanAuthorizationData() {
	cleanCommunityUsersTable()
	cleanEventUsersTable()
	cleanCommunityTable()
	cleanEventsTable()
	cleanUsersTable()
	cleanAddressesTable()
}

func createCommunityOwnedByUserForTest(t *testing.T, name string, owner *domain.UserDomain) string {
	t.Helper()
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}
	community, cErr := communityRepository.CreateCommunity(&domain.CommunityDomain{
		Name:        name,
		Description: "comunidade de teste",
		Owner:       *owner,
		Address:     *address,
	})
	if cErr != nil {
		t.Fatalf("failed to create community: %v", cErr)
	}
	return community.Id
}

func doDelete(t *testing.T, app *fiber.App, url string, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func decodeRestErr(t *testing.T, resp *http.Response) rest_err.RestErr {
	t.Helper()
	defer resp.Body.Close()
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respBody
}

func TestDeleteSkillAuthorization(t *testing.T) {
	t.Cleanup(skillCleanups)

	app := setupApp()
	common := createUserWithRole(t, "skill_auth_user@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "skill_auth_mod@ajuda.dev", domain.UserRoleModerator)
	admin := createUserWithRole(t, "skill_auth_admin@ajuda.dev", domain.UserRoleAdmin)

	skill := registerSkillViaApi(t, app, "go")

	resp := doDelete(t, app, "/v1/skill/"+skill.Id, validTokenFor(t, common.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de skill por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only moderators and admins can delete skills" {
		t.Errorf("esperava message 'only moderators and admins can delete skills', recebeu '%s'", respBody.Message)
	}

	resp = doDelete(t, app, "/v1/skill/"+uuidv7.New().String(), validTokenFor(t, common.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de skill inexistente por USER (autorização vem antes do 404), recebeu %d", resp.StatusCode)
	}

	skillByMod := registerSkillViaApi(t, app, "golang")
	resp = doDelete(t, app, "/v1/skill/"+skillByMod.Id, validTokenFor(t, moderator.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete de skill por MODERATOR, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/skill/"+uuidv7.New().String(), validTokenFor(t, moderator.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no delete de skill inexistente por MODERATOR, recebeu %d", resp.StatusCode)
	}

	skillByAdmin := registerSkillViaApi(t, app, "java")
	resp = doDelete(t, app, "/v1/skill/"+skillByAdmin.Id, validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete de skill por ADMIN, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/skill/"+skill.Id, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de skill sem token, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/skill/"+skill.Id, validTokenFor(t, uuidv7.New().String()))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de skill com usuário do token inexistente no banco, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/skill/abc", validTokenFor(t, moderator.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no delete de skill com id inválido, recebeu %d", resp.StatusCode)
	}
}

func TestDeleteCommunityOnlyOwnerOrStaff(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "com_del_owner@ajuda.dev", domain.UserRoleUser)
	stranger := createUserWithRole(t, "com_del_stranger@ajuda.dev", domain.UserRoleUser)
	communityId := createCommunityOwnedByUserForTest(t, "Comunidade do Dono", owner)

	resp := doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, stranger.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de comunidade por USER não-dono, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the community owner can delete this community" {
		t.Errorf("esperava message 'only the community owner can delete this community', recebeu '%s'", respBody.Message)
	}

	resp = doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, owner.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete da própria comunidade pelo owner, recebeu %d", resp.StatusCode)
	}
	if _, findErr := communityRepository.FindById(communityId); findErr == nil || findErr.Code != fiber.StatusNotFound {
		t.Errorf("esperava comunidade não encontrada após o delete, recebeu %+v", findErr)
	}

	resp = doDelete(t, app, "/v1/community/"+uuidv7.New().String(), validTokenFor(t, stranger.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no delete de comunidade inexistente, recebeu %d", resp.StatusCode)
	}

	softDeletedId := createCommunityOwnedByUserForTest(t, "Comunidade Arquivada", owner)
	if delErr := communityRepository.SoftDeleteById(softDeletedId); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}
	resp = doDelete(t, app, "/v1/community/"+softDeletedId, validTokenFor(t, owner.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no delete de comunidade soft-deletada, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/community/abc", validTokenFor(t, stranger.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no delete de comunidade com id inválido, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/community/"+communityId, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de comunidade sem token, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, uuidv7.New().String()))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de comunidade com usuário do token inexistente no banco, recebeu %d", resp.StatusCode)
	}
}

func TestDeleteCommunityBlockedByMembersUntilTheyLeave(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "com_members_owner@ajuda.dev", domain.UserRoleUser)
	member := createUserWithRole(t, "com_members_member@ajuda.dev", domain.UserRoleUser)
	stranger := createUserWithRole(t, "com_members_stranger@ajuda.dev", domain.UserRoleUser)
	communityId := createCommunityOwnedByUserForTest(t, "Comunidade com Membros", owner)

	joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))

	resp := doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, stranger.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de comunidade com membros por USER não-dono, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no delete de comunidade com membros pelo owner, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "Invalid delete" {
		t.Errorf("esperava message 'Invalid delete', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("members", respBody.Causes)
	if len(causes) == 0 || causes[0] != "community has associated members; remove them before deleting" {
		t.Errorf("esperava cause de membros ativos, recebeu %+v", respBody.Causes)
	}
	if count := membershipCountByCommunity(t, communityId); count != 1 {
		t.Errorf("esperava membership preservada após delete bloqueado, recebeu %d", count)
	}

	respLeave, err := doAuthedRequest(app, newLeaveCommunityRequest(communityId), validTokenFor(t, member.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	respLeave.Body.Close()
	if respLeave.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no leave, recebeu %d", respLeave.StatusCode)
	}

	resp = doDelete(t, app, "/v1/community/"+communityId, validTokenFor(t, owner.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete do owner após os membros saírem, recebeu %d", resp.StatusCode)
	}
}

func TestDeleteCommunityStaffCanDeleteWithMembers(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	moderator := createUserWithRole(t, "com_staff_mod@ajuda.dev", domain.UserRoleModerator)
	admin := createUserWithRole(t, "com_staff_admin@ajuda.dev", domain.UserRoleAdmin)
	owner := createUserWithRole(t, "com_staff_owner@ajuda.dev", domain.UserRoleUser)
	member := createUserWithRole(t, "com_staff_member@ajuda.dev", domain.UserRoleUser)

	communityMod := createCommunityOwnedByUserForTest(t, "Comunidade Staff Mod", owner)
	joinCommunityViaApi(t, app, communityMod, validTokenFor(t, member.Id))
	resp := doDelete(t, app, "/v1/community/"+communityMod, validTokenFor(t, moderator.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete de comunidade com membros por MODERATOR, recebeu %d", resp.StatusCode)
	}

	communityAdmin := createCommunityOwnedByUserForTest(t, "Comunidade Staff Admin", owner)
	joinCommunityViaApi(t, app, communityAdmin, validTokenFor(t, member.Id))
	resp = doDelete(t, app, "/v1/community/"+communityAdmin, validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete de comunidade com membros por ADMIN, recebeu %d", resp.StatusCode)
	}
}

func TestDeleteUserRequiresAdminRole(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	common := createUserWithRole(t, "user_del_common@ajuda.dev", domain.UserRoleUser)
	other := createUserWithRole(t, "user_del_other@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "user_del_mod@ajuda.dev", domain.UserRoleModerator)
	admin := createUserWithRole(t, "user_del_admin@ajuda.dev", domain.UserRoleAdmin)

	resp := doDelete(t, app, "/v1/user/"+other.Id, validTokenFor(t, common.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de usuário por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only admins can delete users" {
		t.Errorf("esperava message 'only admins can delete users', recebeu '%s'", respBody.Message)
	}

	resp = doDelete(t, app, "/v1/user/"+common.Id, validTokenFor(t, moderator.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no delete de usuário por MODERATOR, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/user/"+uuidv7.New().String(), validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no delete de usuário inexistente por ADMIN, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/user/"+common.Id, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de usuário sem token, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/user/"+common.Id, validTokenFor(t, uuidv7.New().String()))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no delete de usuário com usuário do token inexistente no banco, recebeu %d", resp.StatusCode)
	}

	resp = doDelete(t, app, "/v1/user/abc", validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no delete de usuário com id inválido, recebeu %d", resp.StatusCode)
	}

	registerBody := []byte(`{
		"name": "alvo admin",
		"email": "user_del_target@ajuda.dev",
		"password": "123456"
	}`)
	respRegister, err := app.Test(newUserRegisterRequest(registerBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer respRegister.Body.Close()
	if respRegister.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register do alvo, recebeu %d", respRegister.StatusCode)
	}
	var targetDto dto.UserDtoOut
	if err := json.NewDecoder(respRegister.Body).Decode(&targetDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	resp = doDelete(t, app, "/v1/user/"+targetDto.Id, validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete de usuário sem vínculos por ADMIN, recebeu %d", resp.StatusCode)
	}

	loginBody := []byte(`{
		"email": "user_del_target@ajuda.dev",
		"password": "123456"
	}`)
	respLogin, err := app.Test(newUserLoginRequest(loginBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer respLogin.Body.Close()
	if respLogin.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no login do usuário deletado, recebeu %d", respLogin.StatusCode)
	}
}

func TestDeleteUserBlockedByActiveAssociations(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()

	t.Run("owner de comunidade ativa", func(t *testing.T) {
		admin := createUserWithRole(t, "user_links_admin1@ajuda.dev", domain.UserRoleAdmin)
		target := createUserWithRole(t, "user_links_target1@ajuda.dev", domain.UserRoleUser)
		createCommunityOwnedByUserForTest(t, "Comunidade do Alvo 1", target)

		resp := doDelete(t, app, "/v1/user/"+target.Id, validTokenFor(t, admin.Id))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("esperava 400 no delete de dono de comunidade ativa, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		if respBody.Message != "cannot delete user with active associations" {
			t.Errorf("esperava message 'cannot delete user with active associations', recebeu '%s'", respBody.Message)
		}
		causes := getCauseByField("community", respBody.Causes)
		if len(causes) == 0 || causes[0] != "is owner of an active community" {
			t.Errorf("esperava cause do campo 'community', recebeu %+v", respBody.Causes)
		}
	})

	t.Run("owner de evento ativo", func(t *testing.T) {
		admin := createUserWithRole(t, "user_links_admin2@ajuda.dev", domain.UserRoleAdmin)
		target := createUserWithRole(t, "user_links_target2@ajuda.dev", domain.UserRoleUser)
		if _, cErr := eventRepository.CreateEvent(&domain.EventDomain{
			Owner:       *target,
			Category:    domain.CategoryCommunityEvent,
			Type:        domain.TypeOnline,
			Title:       "Evento do Alvo 2",
			Description: "evento ativo",
			StartAt:     time.Now().Add(48 * time.Hour),
			DurationMin: 60,
		}); cErr != nil {
			t.Fatalf("failed to create event: %v", cErr)
		}

		resp := doDelete(t, app, "/v1/user/"+target.Id, validTokenFor(t, admin.Id))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("esperava 400 no delete de dono de evento ativo, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("event", respBody.Causes)
		if len(causes) == 0 || causes[0] != "is owner of an active event" {
			t.Errorf("esperava cause do campo 'event', recebeu %+v", respBody.Causes)
		}
	})

	t.Run("participacao ativa em evento", func(t *testing.T) {
		admin := createUserWithRole(t, "user_links_admin3@ajuda.dev", domain.UserRoleAdmin)
		target := createUserWithRole(t, "user_links_target3@ajuda.dev", domain.UserRoleUser)
		owner := createUserWithRole(t, "user_links_evowner3@ajuda.dev", domain.UserRoleUser)
		event, cErr := eventRepository.CreateEvent(&domain.EventDomain{
			Owner:       *owner,
			Category:    domain.CategoryCommunityEvent,
			Type:        domain.TypeOnline,
			Title:       "Evento do Outro 3",
			Description: "evento ativo",
			StartAt:     time.Now().Add(48 * time.Hour),
			DurationMin: 60,
		})
		if cErr != nil {
			t.Fatalf("failed to create event: %v", cErr)
		}
		if _, pErr := eventUserRepository.CreateOrUpdate(&domain.EventUserDomain{
			EventId: event.Id,
			UserId:  target.Id,
			Role:    domain.RoleAttendee,
			Status:  domain.StatusRequested,
		}, nil); pErr != nil {
			t.Fatalf("failed to create participation: %v", pErr)
		}

		resp := doDelete(t, app, "/v1/user/"+target.Id, validTokenFor(t, admin.Id))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("esperava 400 no delete de usuário com participação ativa, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("event_users", respBody.Causes)
		if len(causes) == 0 || causes[0] != "has an active participation in an event" {
			t.Errorf("esperava cause do campo 'event_users', recebeu %+v", respBody.Causes)
		}
	})

	t.Run("membro de comunidade ativa", func(t *testing.T) {
		admin := createUserWithRole(t, "user_links_admin4@ajuda.dev", domain.UserRoleAdmin)
		target := createUserWithRole(t, "user_links_target4@ajuda.dev", domain.UserRoleUser)
		owner := createUserWithRole(t, "user_links_owner4@ajuda.dev", domain.UserRoleUser)
		communityId := createCommunityOwnedByUserForTest(t, "Comunidade do Outro 4", owner)
		joinCommunityViaApi(t, app, communityId, validTokenFor(t, target.Id))

		resp := doDelete(t, app, "/v1/user/"+target.Id, validTokenFor(t, admin.Id))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("esperava 400 no delete de membro de comunidade ativa, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("community_users", respBody.Causes)
		if len(causes) == 0 || causes[0] != "is a member of an active community" {
			t.Errorf("esperava cause do campo 'community_users', recebeu %+v", respBody.Causes)
		}
	})
}

func TestCommunityRegisterUsesAuthenticatedUserAsOwner(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	userA := createUserWithRole(t, "reg_com_a@ajuda.dev", domain.UserRoleUser)
	userB := createUserWithRole(t, "reg_com_b@ajuda.dev", domain.UserRoleUser)
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}

	body := []byte(`{
		"address_id": "` + address.Id + `",
		"owner_id": "` + userB.Id + `",
		"name": "Comunidade Token Owner",
		"description": "owner deve vir do token"
	}`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, validTokenFor(t, userA.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register com owner_id de outro usuário, recebeu %d", resp.StatusCode)
	}
	var respDto dto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	resp.Body.Close()
	if respDto.Owner == nil || respDto.Owner.Id != userA.Id {
		t.Errorf("esperava owner.id '%s' (do token), recebeu %+v", userA.Id, respDto.Owner)
	}

	body = []byte(`{
		"address_id": "` + address.Id + `",
		"name": "Comunidade Sem Owner",
		"description": "owner deve vir do token"
	}`)
	req = newCommunityRegisterRequest(body)
	resp, err = doAuthedRequest(app, req, validTokenFor(t, userA.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register sem owner_id, recebeu %d", resp.StatusCode)
	}
	respDto = dto.CommunityDto{}
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	resp.Body.Close()
	if respDto.Owner == nil || respDto.Owner.Id != userA.Id {
		t.Errorf("esperava owner.id '%s' (do token), recebeu %+v", userA.Id, respDto.Owner)
	}
}

func TestEventRegisterUsesAuthenticatedUserAsOwner(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	userA := createUserWithRole(t, "reg_event_a@ajuda.dev", domain.UserRoleUser)
	userB := createUserWithRole(t, "reg_event_b@ajuda.dev", domain.UserRoleUser)

	payload, err := json.Marshal(eventTestRequest{
		OwnerId:     userB.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento Token Owner",
		Description: "owner deve vir do token",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, userA.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register com owner_id de outro usuário, recebeu %d", resp.StatusCode)
	}
	var respDto dto.RegisterEventDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	resp.Body.Close()
	if respDto.OwnerId != userA.Id {
		t.Errorf("esperava owner_id '%s' (do token), recebeu '%s'", userA.Id, respDto.OwnerId)
	}

	payload, err = json.Marshal(eventTestRequest{
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento Sem Owner",
		Description: "owner deve vir do token",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err = doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, userA.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register sem owner_id, recebeu %d", resp.StatusCode)
	}
	respDto = dto.RegisterEventDto{}
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	resp.Body.Close()
	if respDto.OwnerId != userA.Id {
		t.Errorf("esperava owner_id '%s' (do token), recebeu '%s'", userA.Id, respDto.OwnerId)
	}
}

func TestRegisterAndLoginReturnRole(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()

	registerBody := []byte(`{
		"name": "novo usuario",
		"email": "reg_role@ajuda.dev",
		"password": "123456"
	}`)
	resp, err := app.Test(newUserRegisterRequest(registerBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no register, recebeu %d", resp.StatusCode)
	}
	var registerDto dto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&registerDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if registerDto.Role != domain.UserRoleUser {
		t.Errorf("esperava role 'USER' na resposta do register, recebeu '%s'", registerDto.Role)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if _, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "moderador",
		Email:    "role_mod@ajuda.dev",
		Password: string(hashedPassword),
		Role:     domain.UserRoleModerator,
	}); createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	loginBody := []byte(`{
		"email": "role_mod@ajuda.dev",
		"password": "123456"
	}`)
	respLogin, err := app.Test(newUserLoginRequest(loginBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer respLogin.Body.Close()
	if respLogin.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no login, recebeu %d", respLogin.StatusCode)
	}
	var loginDto dto.LoginUserDtoOut
	if err := json.NewDecoder(respLogin.Body).Decode(&loginDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if loginDto.Role != domain.UserRoleModerator {
		t.Errorf("esperava role 'MODERATOR' na resposta do login, recebeu '%s'", loginDto.Role)
	}
}
