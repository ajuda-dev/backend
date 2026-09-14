package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
)

func newGetUserCommunitiesRequest(userId string, query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/v1/user/"+userId+"/communities"+query, nil)
}

func getUserCommunitiesViaApi(t *testing.T, app *fiber.App, userId string, query string, token string) dto.PageableCommunityDto {
	t.Helper()
	resp, err := doAuthedRequest(app, newGetUserCommunitiesRequest(userId, query), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 200 ao listar comunidades do usuário, recebeu %d (body: %s)", resp.StatusCode, string(body))
	}
	var respDto dto.PageableCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func TestGetUserCommunitiesSuccess(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	second := createCommunityOwnedByForTest(t, "Zulu Comunidade", "owner_user_communities2@ajuda.dev")
	first := createCommunityOwnedByForTest(t, "Alfa Comunidade", "owner_user_communities1@ajuda.dev")
	member := createCommunityUserForTest(t, "member_user_communities@ajuda.dev")
	token := validTokenFor(t, member.Id)
	joinCommunityViaApi(t, app, second, token)
	joinCommunityViaApi(t, app, first, token)

	respDto := getUserCommunitiesViaApi(t, app, member.Id, "", token)

	if len(respDto.Data) != 2 {
		t.Fatalf("esperava 2 comunidades, recebeu %d", len(respDto.Data))
	}
	if respDto.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}
	if respDto.Data[0].Id != first {
		t.Errorf("esperava a primeira comunidade ordenada por nome (alfa), recebeu %s", respDto.Data[0].Id)
	}
	if respDto.Data[1].Id != second {
		t.Errorf("esperava a segunda comunidade ordenada por nome (zulu), recebeu %s", respDto.Data[1].Id)
	}
	for _, community := range respDto.Data {
		if community.Name == "" {
			t.Errorf("esperava name preenchido")
		}
		if community.Address == nil {
			t.Fatalf("esperava address embutido na comunidade %s", community.Id)
		}
		if community.Address.City == "" {
			t.Errorf("esperava address.city preenchido")
		}
		if community.Owner == nil {
			t.Fatalf("esperava owner embutido na comunidade %s", community.Id)
		}
		if community.Owner.Id == "" {
			t.Errorf("esperava owner.id preenchido")
		}
	}
}

func TestGetUserCommunitiesExcludesSoftDeletedCommunity(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	active := createCommunityOwnedByForTest(t, "Ativa User Communities", "owner_active_user_communities@ajuda.dev")
	deleted := createCommunityOwnedByForTest(t, "Deletada User Communities", "owner_deleted_user_communities@ajuda.dev")
	member := createCommunityUserForTest(t, "member_deleted_user_communities@ajuda.dev")
	token := validTokenFor(t, member.Id)
	joinCommunityViaApi(t, app, active, token)
	joinCommunityViaApi(t, app, deleted, token)

	if err := communityRepository.SoftDeleteById(deleted); err != nil {
		t.Fatalf("failed to soft delete community: %v", err)
	}

	respDto := getUserCommunitiesViaApi(t, app, member.Id, "", token)

	if len(respDto.Data) != 1 {
		t.Fatalf("esperava 1 comunidade após o soft delete, recebeu %d", len(respDto.Data))
	}
	if respDto.Data[0].Id != active {
		t.Errorf("esperava a comunidade ativa na lista, recebeu %s", respDto.Data[0].Id)
	}
}

func TestGetUserCommunitiesWithoutMembership(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	user := createCommunityUserForTest(t, "no_membership_user_communities@ajuda.dev")

	respDto := getUserCommunitiesViaApi(t, app, user.Id, "", validTokenFor(t, user.Id))

	if len(respDto.Data) != 0 {
		t.Errorf("esperava lista vazia, recebeu %d itens", len(respDto.Data))
	}
	if respDto.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}
}

func TestGetUserCommunitiesIgnoresOwnedCommunities(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	owner := createUserWithRole(t, "owner_owned_user_communities@ajuda.dev", "USER")
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}
	if _, cErr := communityRepository.CreateCommunity(&domain.CommunityDomain{
		Name:        "Comunidade Própria",
		Description: "comunidade de teste",
		Owner:       *owner,
		Address:     *address,
	}); cErr != nil {
		t.Fatalf("failed to create community: %v", cErr)
	}

	respDto := getUserCommunitiesViaApi(t, app, owner.Id, "", validTokenFor(t, owner.Id))

	if len(respDto.Data) != 0 {
		t.Errorf("esperava lista vazia (dono não possui membership), recebeu %d itens", len(respDto.Data))
	}
	if respDto.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}
}

func TestGetUserCommunitiesAnyAuthenticatedUserCanRead(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	communityId := createCommunityOwnedByForTest(t, "Comunidade Leitura Livre", "owner_free_read_user_communities@ajuda.dev")
	target := createCommunityUserForTest(t, "target_free_read_user_communities@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, target.Id))
	requester := createCommunityUserForTest(t, "requester_free_read_user_communities@ajuda.dev")

	respDto := getUserCommunitiesViaApi(t, app, target.Id, "", validTokenFor(t, requester.Id))

	if len(respDto.Data) != 1 {
		t.Fatalf("esperava 1 comunidade do alvo, recebeu %d", len(respDto.Data))
	}
	if respDto.Data[0].Id != communityId {
		t.Errorf("esperava a comunidade %s, recebeu %s", communityId, respDto.Data[0].Id)
	}
}

func TestGetUserCommunitiesPagination(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	first := createCommunityOwnedByForTest(t, "Alfa Paged User Communities", "owner_paged_user_communities1@ajuda.dev")
	second := createCommunityOwnedByForTest(t, "Bravo Paged User Communities", "owner_paged_user_communities2@ajuda.dev")
	third := createCommunityOwnedByForTest(t, "Charlie Paged User Communities", "owner_paged_user_communities3@ajuda.dev")
	member := createCommunityUserForTest(t, "member_paged_user_communities@ajuda.dev")
	token := validTokenFor(t, member.Id)
	joinCommunityViaApi(t, app, first, token)
	joinCommunityViaApi(t, app, second, token)
	joinCommunityViaApi(t, app, third, token)

	firstPage := getUserCommunitiesViaApi(t, app, member.Id, "?limit=2", token)
	if len(firstPage.Data) != 2 {
		t.Fatalf("esperava 2 itens na primeira página, recebeu %d", len(firstPage.Data))
	}
	if !firstPage.HasNext {
		t.Errorf("esperava has_next true na primeira página")
	}

	secondPage := getUserCommunitiesViaApi(t, app, member.Id, "?page=2&limit=2", token)
	if len(secondPage.Data) != 1 {
		t.Fatalf("esperava 1 item na segunda página, recebeu %d", len(secondPage.Data))
	}
	if secondPage.HasNext {
		t.Errorf("esperava has_next false na segunda página")
	}
	if secondPage.Data[0].Id != third {
		t.Errorf("esperava a terceira comunidade na segunda página, recebeu %s", secondPage.Data[0].Id)
	}
}

func TestGetUserCommunitiesRejectsInvalidId(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	requester := createCommunityUserForTest(t, "requester_invalid_user_communities@ajuda.dev")
	resp, err := doAuthedRequest(app, newGetUserCommunitiesRequest("abc", ""), validTokenFor(t, requester.Id))
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
	messages := getCauseByField("userId", respBody.Causes)
	if len(messages) != 1 || messages[0] != "userId must be a valid UUID v7" {
		t.Errorf("esperava causa 'userId must be a valid UUID v7', recebeu %v", messages)
	}
}

func TestGetUserCommunitiesRequiresToken(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)
	app := setupApp()

	user := createCommunityUserForTest(t, "no_token_user_communities@ajuda.dev")
	resp, err := app.Test(newGetUserCommunitiesRequest(user.Id, ""))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
}
