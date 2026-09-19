package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func hideUserEmail(t *testing.T, app *fiber.App, user *userdomain.UserDomain) {
	t.Helper()
	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"email": {"shareWithCommunity": false}}}`), validTokenFor(t, user.Id))
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao esconder o e-mail, recebeu %d", resp.StatusCode)
	}
}

func decodeJSONObject(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return body
}

func communityOwnerFromJSON(t *testing.T, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	owner, ok := body["owner"].(map[string]interface{})
	if !ok || owner == nil {
		t.Fatalf("esperava owner objeto, recebeu %+v", body["owner"])
	}
	return owner
}

func assertCommunityOwnerPIIHidden(t *testing.T, owner map[string]interface{}) {
	t.Helper()
	if email, exists := owner["email"]; exists && email != "" {
		t.Errorf("esperava owner.email ausente, recebeu %v", email)
	}
	if _, exists := owner["role"]; exists {
		t.Errorf("esperava owner.role ausente, recebeu %v", owner["role"])
	}
	if _, exists := owner["emailVerified"]; exists {
		t.Errorf("esperava owner.emailVerified ausente, recebeu %v", owner["emailVerified"])
	}
}

func assertCommunityOwnerEmailVisible(t *testing.T, owner map[string]interface{}, email string) {
	t.Helper()
	if owner["email"] != email {
		t.Errorf("esperava owner.email %s, recebeu %v", email, owner["email"])
	}
	if _, exists := owner["role"]; exists {
		t.Errorf("esperava owner.role ausente, recebeu %v", owner["role"])
	}
	if _, exists := owner["emailVerified"]; exists {
		t.Errorf("esperava owner.emailVerified ausente, recebeu %v", owner["emailVerified"])
	}
}

func assertGhostTokenUnauthorized(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 para token fantasma, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func setupPrivateOwnerCommunity(t *testing.T, app *fiber.App) (*userdomain.UserDomain, string) {
	t.Helper()
	owner := createUserWithRole(t, "owner.visibility.community@ajuda.dev", userdomain.UserRoleUser)
	address := createCommunityAddress(t, "owner-visibility-city")
	communityId := registerCommunityForOwner(t, app, validTokenFor(t, owner.Id), address.Id, "Comunidade Owner Visibility")
	hideUserEmail(t, app, owner)
	return owner, communityId
}

func TestGetCommunityByIdHidesOwnerEmailFromThirdParty(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	owner, communityId := setupPrivateOwnerCommunity(t, app)
	third := createUserWithRole(t, "third.visibility.community@ajuda.dev", userdomain.UserRoleUser)

	resp := getCommunityByIdRequest(t, app, validTokenFor(t, third.Id), communityId)
	if resp.StatusCode != fiber.StatusOK {
		resp.Body.Close()
		t.Fatalf("esperava 200 no GET por id para terceiro, recebeu %d", resp.StatusCode)
	}
	ownerJSON := communityOwnerFromJSON(t, decodeJSONObject(t, resp))
	if ownerJSON["id"] != owner.Id {
		t.Errorf("esperava owner.id %s, recebeu %v", owner.Id, ownerJSON["id"])
	}
	assertCommunityOwnerPIIHidden(t, ownerJSON)
}

func TestGetCommunityByIdShowsOwnerEmailToOwner(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	owner, communityId := setupPrivateOwnerCommunity(t, app)

	resp := getCommunityByIdRequest(t, app, validTokenFor(t, owner.Id), communityId)
	if resp.StatusCode != fiber.StatusOK {
		resp.Body.Close()
		t.Fatalf("esperava 200 no GET por id para o dono, recebeu %d", resp.StatusCode)
	}
	assertCommunityOwnerEmailVisible(t, communityOwnerFromJSON(t, decodeJSONObject(t, resp)), owner.Email)
}

func TestGetCommunityByIdShowsOwnerEmailToAdmin(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	owner, communityId := setupPrivateOwnerCommunity(t, app)
	admin := createUserWithRole(t, "admin.visibility.community@ajuda.dev", userdomain.UserRoleAdmin)

	resp := getCommunityByIdRequest(t, app, validTokenFor(t, admin.Id), communityId)
	if resp.StatusCode != fiber.StatusOK {
		resp.Body.Close()
		t.Fatalf("esperava 200 no GET por id para o admin, recebeu %d", resp.StatusCode)
	}
	assertCommunityOwnerEmailVisible(t, communityOwnerFromJSON(t, decodeJSONObject(t, resp)), owner.Email)
}

func TestListCommunitiesHidesOwnerEmailFromThirdParty(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	owner, communityId := setupPrivateOwnerCommunity(t, app)
	third := createUserWithRole(t, "third.list.visibility.community@ajuda.dev", userdomain.UserRoleUser)

	req := httptest.NewRequest(http.MethodGet, "/v1/community?owner_id="+owner.Id, nil)
	resp, err := doAuthedRequest(app, req, validTokenFor(t, third.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		resp.Body.Close()
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	body := decodeJSONObject(t, resp)
	data, ok := body["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("esperava 1 comunidade na listagem, recebeu %+v", body["data"])
	}
	item, ok := data[0].(map[string]interface{})
	if !ok || item["id"] != communityId {
		t.Fatalf("esperava a comunidade %s, recebeu %+v", communityId, data[0])
	}
	assertCommunityOwnerPIIHidden(t, communityOwnerFromJSON(t, item))
}

func TestGetUserCommunitiesHidesOwnerEmailFromThirdParty(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	_, communityId := setupPrivateOwnerCommunity(t, app)
	member := createCommunityUserForTest(t, "member.visibility.user.communities@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))
	third := createUserWithRole(t, "third.visibility.user.communities@ajuda.dev", userdomain.UserRoleUser)

	req := newGetUserCommunitiesRequest(member.Id, "")
	resp, err := doAuthedRequest(app, req, validTokenFor(t, third.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		resp.Body.Close()
		t.Fatalf("esperava 200 na listagem de comunidades do membro, recebeu %d", resp.StatusCode)
	}
	body := decodeJSONObject(t, resp)
	data, ok := body["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("esperava 1 comunidade do membro, recebeu %+v", body["data"])
	}
	item, ok := data[0].(map[string]interface{})
	if !ok || item["id"] != communityId {
		t.Fatalf("esperava a comunidade %s, recebeu %+v", communityId, data[0])
	}
	assertCommunityOwnerPIIHidden(t, communityOwnerFromJSON(t, item))
}

func TestCommunityOwnerReadsRejectGhostToken(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	_, communityId := setupPrivateOwnerCommunity(t, app)
	member := createCommunityUserForTest(t, "member.ghost.user.communities@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))
	ghost := validTokenFor(t, uuidv7.New().String())

	detail, err := doAuthedRequest(app, httptest.NewRequest(http.MethodGet, "/v1/community/"+communityId, nil), ghost)
	if err != nil {
		t.Fatalf("erro no GET por id: %v", err)
	}
	assertGhostTokenUnauthorized(t, detail)

	list, err := doAuthedRequest(app, httptest.NewRequest(http.MethodGet, "/v1/community", nil), ghost)
	if err != nil {
		t.Fatalf("erro no GET da listagem: %v", err)
	}
	assertGhostTokenUnauthorized(t, list)

	memberships, err := doAuthedRequest(app, newGetUserCommunitiesRequest(member.Id, ""), ghost)
	if err != nil {
		t.Fatalf("erro no GET de comunidades do usuário: %v", err)
	}
	assertGhostTokenUnauthorized(t, memberships)
}

func TestGetCommunityMembersStillHidesUndeclaredEmail(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	_, communityId := setupPrivateOwnerCommunity(t, app)
	hidden := createCommunityUserForTest(t, "hidden.regression.members@ajuda.dev")
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, hidden.Id))
	requester := createCommunityUserForTest(t, "requester.regression.members@ajuda.dev")

	resp, err := doAuthedRequest(app, newGetCommunityMembersRequest(communityId, ""), validTokenFor(t, requester.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem de membros, recebeu %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	if strings.Contains(string(raw), hidden.Email) {
		t.Errorf("esperava o e-mail não compartilhado fora do body, recebeu %s", raw)
	}
}
