package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
)

func newGetCommunityMembersRequest(communityId string, query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/v1/community/"+communityId+"/members"+query, nil)
}

func getCommunityMembersViaApi(t *testing.T, app *fiber.App, communityId string, query string, token string) dto.PageableCommunityMemberDto {
	t.Helper()
	resp, err := doAuthedRequest(app, newGetCommunityMembersRequest(communityId, query), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 200 ao listar membros, recebeu %d (body: %s)", resp.StatusCode, string(body))
	}
	var respDto dto.PageableCommunityMemberDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func TestGetCommunityMembersSuccess(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade membros", "owner_members@ajuda.dev")
	bruno := createNamedCommunityUserForTest(t, "Bruno Sá", "bruno_members@ajuda.dev")
	ana := createNamedCommunityUserForTest(t, "Ana Lima", "ana_members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, bruno.Id))
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, ana.Id))

	respDto := getCommunityMembersViaApi(t, app, communityId, "", validTokenFor(t, ana.Id))

	if len(respDto.Data) != 2 {
		t.Fatalf("esperava 2 membros (o dono não possui membership), recebeu %d", len(respDto.Data))
	}
	if respDto.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}
	if respDto.Data[0].UserId != ana.Id {
		t.Errorf("esperava o primeiro item ordenado por nome (ana), recebeu %s", respDto.Data[0].UserId)
	}
	if respDto.Data[1].UserId != bruno.Id {
		t.Errorf("esperava o segundo item ordenado por nome (bruno), recebeu %s", respDto.Data[1].UserId)
	}
	for _, member := range respDto.Data {
		if member.Id == "" {
			t.Errorf("esperava id da membership preenchido")
		}
		if member.CommunityId != communityId {
			t.Errorf("esperava community_id %s, recebeu %s", communityId, member.CommunityId)
		}
		if member.User == nil {
			t.Fatalf("esperava usuário embutido no item da membership %s", member.Id)
		}
		if member.User.Id != member.UserId {
			t.Errorf("esperava user.id %s, recebeu %s", member.UserId, member.User.Id)
		}
		if member.User.Name == "" {
			t.Errorf("esperava user.name preenchido")
		}
	}
}

func TestGetCommunityMembersWithoutMembers(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade vazia", "owner_empty_members@ajuda.dev")
	requester := createCommunityUserForTest(t, "requester_empty_members@ajuda.dev")

	respDto := getCommunityMembersViaApi(t, app, communityId, "", validTokenFor(t, requester.Id))

	if len(respDto.Data) != 0 {
		t.Errorf("esperava lista vazia, recebeu %d itens", len(respDto.Data))
	}
	if respDto.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}
}

func TestGetCommunityMembersPagination(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade paginada", "owner_paged_members@ajuda.dev")
	first := createNamedCommunityUserForTest(t, "Ana Paged", "ana_paged_members@ajuda.dev")
	second := createNamedCommunityUserForTest(t, "Bruno Paged", "bruno_paged_members@ajuda.dev")
	third := createNamedCommunityUserForTest(t, "Carla Paged", "carla_paged_members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, first.Id))
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, second.Id))
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, third.Id))

	firstPage := getCommunityMembersViaApi(t, app, communityId, "?limit=2", validTokenFor(t, first.Id))
	if len(firstPage.Data) != 2 {
		t.Fatalf("esperava 2 itens na primeira página, recebeu %d", len(firstPage.Data))
	}
	if !firstPage.HasNext {
		t.Errorf("esperava has_next true na primeira página")
	}

	secondPage := getCommunityMembersViaApi(t, app, communityId, "?page=2&limit=2", validTokenFor(t, first.Id))
	if len(secondPage.Data) != 1 {
		t.Fatalf("esperava 1 item na segunda página, recebeu %d", len(secondPage.Data))
	}
	if secondPage.HasNext {
		t.Errorf("esperava has_next false na segunda página")
	}
	if secondPage.Data[0].UserId != third.Id {
		t.Errorf("esperava o terceiro membro na segunda página, recebeu %s", secondPage.Data[0].UserId)
	}
}

func TestGetCommunityMembersHidesUndeclaredEmail(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade email", "owner_email_members@ajuda.dev")
	hidden := createCommunityUserForTest(t, "hidden_email_members@ajuda.dev")
	shared := createCommunityUserForTest(t, "shared_email_members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, hidden.Id))
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, shared.Id))

	_, updateErr := userRepository.Update(shared.Id, &domain.UserDomain{
		ConfigVisibility: domain.ConfigVisibility{
			domain.VisibilityKeyEmail: {Value: shared.Email, ShareWithCommunity: true},
		},
	})
	if updateErr != nil {
		t.Fatalf("failed to update visibility: %v", updateErr)
	}

	requester := createCommunityUserForTest(t, "requester_email_members@ajuda.dev")
	resp, err := doAuthedRequest(app, newGetCommunityMembersRequest(communityId, ""), validTokenFor(t, requester.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	body := string(rawBody)

	if strings.Contains(body, hidden.Email) {
		t.Errorf("esperava o e-mail não compartilhado fora do body, recebeu %s", body)
	}
	if !strings.Contains(body, shared.Email) {
		t.Errorf("esperava o e-mail compartilhado no body, recebeu %s", body)
	}

	ownView := getCommunityMembersViaApi(t, app, communityId, "", validTokenFor(t, hidden.Id))
	for _, member := range ownView.Data {
		if member.UserId != hidden.Id {
			continue
		}
		if member.User == nil || member.User.Email != hidden.Email {
			t.Errorf("esperava o próprio e-mail visível para o membro, recebeu %+v", member.User)
		}
	}
}

func TestGetCommunityMembersExcludesSoftDeletedUser(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade deletado", "owner_deleted_members@ajuda.dev")
	active := createCommunityUserForTest(t, "active_deleted_members@ajuda.dev")
	deleted := createCommunityUserForTest(t, "deleted_members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, active.Id))
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, deleted.Id))

	if err := userRepository.SoftDeleteById(deleted.Id); err != nil {
		t.Fatalf("failed to soft delete user: %v", err)
	}

	respDto := getCommunityMembersViaApi(t, app, communityId, "", validTokenFor(t, active.Id))

	if len(respDto.Data) != 1 {
		t.Fatalf("esperava 1 membro após o soft delete, recebeu %d", len(respDto.Data))
	}
	if respDto.Data[0].UserId != active.Id {
		t.Errorf("esperava o membro ativo na lista, recebeu %s", respDto.Data[0].UserId)
	}
}

func TestGetCommunityMembersRejectsInvalidId(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	requester := createCommunityUserForTest(t, "requester_invalid_members@ajuda.dev")
	resp, err := doAuthedRequest(app, newGetCommunityMembersRequest("abc", ""), validTokenFor(t, requester.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para id inválido, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Err != "bad_request" {
		t.Errorf("esperava error 'bad_request', recebeu '%s'", respBody.Err)
	}
	messages := getCauseByField("id", respBody.Causes)
	if len(messages) != 1 || messages[0] != "id must be a valid UUID v7" {
		t.Errorf("esperava causa 'id must be a valid UUID v7', recebeu %v", messages)
	}
}

func TestGetCommunityMembersNotFoundForDeletedCommunity(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade removida", "owner_removed_members@ajuda.dev")
	member := createCommunityUserForTest(t, "member_removed_members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))

	if err := communityRepository.SoftDeleteById(communityId); err != nil {
		t.Fatalf("failed to soft delete community: %v", err)
	}

	resp, err := doAuthedRequest(app, newGetCommunityMembersRequest(communityId, ""), validTokenFor(t, member.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para comunidade soft-deletada, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "community not found" {
		t.Errorf("esperava mensagem 'community not found', recebeu '%s'", respBody.Message)
	}
}

func TestGetCommunityMembersRequiresToken(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "comunidade sem token", "owner_no_token_members@ajuda.dev")
	resp, err := app.Test(newGetCommunityMembersRequest(communityId, ""))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
}
