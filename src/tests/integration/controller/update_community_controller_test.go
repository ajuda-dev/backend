package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newUpdateCommunityRequest(id string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/v1/community/"+id, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func updateCommunityViaApi(t *testing.T, app *fiber.App, id string, token string, body []byte) *http.Response {
	t.Helper()
	resp, err := doAuthedRequest(app, newUpdateCommunityRequest(id, body), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func decodeCommunityDto(t *testing.T, resp *http.Response) dto.CommunityDto {
	t.Helper()
	defer resp.Body.Close()
	var community dto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&community); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return community
}

func getCommunityById(t *testing.T, app *fiber.App, id string, token string) dto.CommunityDto {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/community/"+id, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET, recebeu %d", resp.StatusCode)
	}
	var community dto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&community); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return community
}

func TestUpdateCommunityOwnerChangesName(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.name@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Nome")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Renomeada"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Id != communityId || updated.Name != "Comunidade Renomeada" {
		t.Errorf("esperava id '%s' com name 'Comunidade Renomeada', recebeu %+v", communityId, updated)
	}
	if updated.Description != "comunidade de teste" {
		t.Errorf("descrição não informada não deveria mudar, recebeu '%s'", updated.Description)
	}
	if updated.Owner == nil || updated.Owner.Id != owner.Id {
		t.Errorf("esperava owner '%s' na resposta, recebeu %+v", owner.Id, updated.Owner)
	}
	if updated.Address == nil || updated.Address.Id != address.Id {
		t.Errorf("esperava address '%s' na resposta, recebeu %+v", address.Id, updated.Address)
	}

	persisted := getCommunityById(t, app, communityId, token)
	if persisted.Name != "Comunidade Renomeada" {
		t.Errorf("esperava name persistido 'Comunidade Renomeada', recebeu '%s'", persisted.Name)
	}
}

func TestUpdateCommunityOwnerChangesDescription(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.description@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Descricao")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"description":"nova descricao"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Description != "nova descricao" {
		t.Errorf("esperava description 'nova descricao', recebeu '%s'", updated.Description)
	}
	if updated.Name != "Comunidade Update Descricao" {
		t.Errorf("nome não informado não deveria mudar, recebeu '%s'", updated.Name)
	}

	persisted := getCommunityById(t, app, communityId, token)
	if persisted.Description != "nova descricao" {
		t.Errorf("esperava description persistida 'nova descricao', recebeu '%s'", persisted.Description)
	}
}

func TestUpdateCommunityOwnerChangesAddress(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.address@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	oldAddress := createCommunityAddress(t, "sao paulo")
	newAddress := createCommunityAddress(t, "campinas")
	communityId := registerCommunityViaApi(t, app, token, oldAddress.Id, "Comunidade Update Endereco")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"address_id":"`+newAddress.Id+`"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Address == nil || updated.Address.Id != newAddress.Id {
		t.Fatalf("esperava address.id '%s' na resposta, recebeu %+v", newAddress.Id, updated.Address)
	}
	if updated.Address.City != "campinas" {
		t.Errorf("esperava address.city 'campinas' na resposta, recebeu '%s'", updated.Address.City)
	}

	persisted := getCommunityById(t, app, communityId, token)
	if persisted.Address == nil || persisted.Address.Id != newAddress.Id {
		t.Errorf("esperava address.id '%s' persistido, recebeu %+v", newAddress.Id, persisted.Address)
	}
}

func TestUpdateCommunityOwnerWithActiveMembers(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.members@ajuda.dev")
	member := createCommunityOwner(t, "member.update.members@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Com Membros")

	joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Com Membros Renomeada"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("dono deve alterar mesmo com membros ativos: esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Name != "Comunidade Com Membros Renomeada" {
		t.Errorf("esperava name 'Comunidade Com Membros Renomeada', recebeu '%s'", updated.Name)
	}
}

func TestUpdateCommunityNonOwnerForbidden(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.forbidden@ajuda.dev")
	other := createCommunityOwner(t, "other.update.forbidden@ajuda.dev")
	ownerToken := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, ownerToken, address.Id, "Comunidade Update Proibida")

	resp := updateCommunityViaApi(t, app, communityId, validTokenFor(t, other.Id), []byte(`{"name":"Nome Do Outro"}`))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the community owner can update this community" {
		t.Errorf("esperava message 'only the community owner can update this community', recebeu '%s'", respBody.Message)
	}

	persisted := getCommunityById(t, app, communityId, ownerToken)
	if persisted.Name != "Comunidade Update Proibida" {
		t.Errorf("a comunidade não deveria ter mudado, recebeu name '%s'", persisted.Name)
	}
}

func TestUpdateCommunityModeratorAndAdminAllowed(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.staff@ajuda.dev")
	moderator := createUserWithRole(t, "moderator.update.staff@ajuda.dev", domain.UserRoleModerator)
	admin := createUserWithRole(t, "admin.update.staff@ajuda.dev", domain.UserRoleAdmin)
	ownerToken := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")

	moderatorCommunity := registerCommunityViaApi(t, app, ownerToken, address.Id, "Comunidade Update Moderador")
	resp := updateCommunityViaApi(t, app, moderatorCommunity, validTokenFor(t, moderator.Id), []byte(`{"name":"Comunidade Renomeada Por Moderador"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("moderador não-dono: esperava 200, recebeu %d", resp.StatusCode)
	}
	if updated := decodeCommunityDto(t, resp); updated.Name != "Comunidade Renomeada Por Moderador" {
		t.Errorf("esperava name 'Comunidade Renomeada Por Moderador', recebeu '%s'", updated.Name)
	}

	adminCommunity := registerCommunityViaApi(t, app, ownerToken, address.Id, "Comunidade Update Admin")
	resp = updateCommunityViaApi(t, app, adminCommunity, validTokenFor(t, admin.Id), []byte(`{"name":"Comunidade Renomeada Por Admin"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("admin não-dono: esperava 200, recebeu %d", resp.StatusCode)
	}
	if updated := decodeCommunityDto(t, resp); updated.Name != "Comunidade Renomeada Por Admin" {
		t.Errorf("esperava name 'Comunidade Renomeada Por Admin', recebeu '%s'", updated.Name)
	}
}

func TestUpdateCommunityValidationErrors(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.validation@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Validacao")

	t.Run("body vazio", func(t *testing.T) {
		resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{}`))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		if respBody.Message != "Invalid community data" {
			t.Errorf("esperava message 'Invalid community data', recebeu '%s'", respBody.Message)
		}
		causes := getCauseByField("body", respBody.Causes)
		if len(causes) == 0 || causes[0] != "provide at least one field to update" {
			t.Errorf("esperava cause do campo 'body', recebeu %+v", respBody.Causes)
		}
	})

	t.Run("somente espacos", func(t *testing.T) {
		resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"   "}`))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("body", respBody.Causes)
		if len(causes) == 0 || causes[0] != "provide at least one field to update" {
			t.Errorf("esperava cause do campo 'body', recebeu %+v", respBody.Causes)
		}
	})

	t.Run("address_id nao uuid", func(t *testing.T) {
		resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"address_id":"abc"}`))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("address_id", respBody.Causes)
		if len(causes) == 0 || causes[0] != "AddressId is not valid" {
			t.Errorf("esperava cause do campo 'address_id', recebeu %+v", respBody.Causes)
		}
	})
}

func TestUpdateCommunityNameAlreadyUsed(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.duplicate@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Duplicada")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Alvo")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Update Duplicada"}`))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "Invalid community data" {
		t.Errorf("esperava message 'Invalid community data', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "already has a community with this name" {
		t.Errorf("esperava cause 'already has a community with this name', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateCommunitySameNameIsAllowed(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.samename@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Mesmo Nome")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Update Mesmo Nome"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("reenviar o próprio nome não pode acusar duplicidade: esperava 200, recebeu %d", resp.StatusCode)
	}
	if updated := decodeCommunityDto(t, resp); updated.Name != "Comunidade Update Mesmo Nome" {
		t.Errorf("esperava name 'Comunidade Update Mesmo Nome', recebeu '%s'", updated.Name)
	}
}

func TestUpdateCommunityNameOfSoftDeletedCommunityIsReusable(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.softdeleted@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	archivedId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Arquivada")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Reuso")

	req := httptest.NewRequest(http.MethodDelete, "/v1/community/"+archivedId, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 ao arquivar, recebeu %d", resp.StatusCode)
	}

	resp = updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Update Arquivada"}`))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("nome de comunidade arquivada é reutilizável: esperava 200, recebeu %d", resp.StatusCode)
	}
	if updated := decodeCommunityDto(t, resp); updated.Name != "Comunidade Update Arquivada" {
		t.Errorf("esperava name 'Comunidade Update Arquivada', recebeu '%s'", updated.Name)
	}
}

func TestUpdateCommunityAddressNotFound(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.address404@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Endereco Invalido")

	resp := updateCommunityViaApi(t, app, communityId, token, []byte(`{"address_id":"`+uuidv7.New().String()+`"}`))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "Invalid community data" {
		t.Errorf("esperava message 'Invalid community data', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("address_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "address_id is not valid, not found this address" {
		t.Errorf("esperava cause 'address_id is not valid, not found this address', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateCommunityNotFound(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.notfound@ajuda.dev")
	token := validTokenFor(t, owner.Id)

	resp := updateCommunityViaApi(t, app, uuidv7.New().String(), token, []byte(`{"name":"Comunidade Inexistente"}`))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "community not found" {
		t.Errorf("esperava message 'community not found', recebeu '%s'", respBody.Message)
	}
}

func TestUpdateCommunityArchivedNotFound(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.archived@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Arquivada 404")

	req := httptest.NewRequest(http.MethodDelete, "/v1/community/"+communityId, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 ao arquivar, recebeu %d", resp.StatusCode)
	}

	resp = updateCommunityViaApi(t, app, communityId, token, []byte(`{"name":"Comunidade Arquivada Renomeada"}`))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para comunidade arquivada, recebeu %d", resp.StatusCode)
	}
}

func TestUpdateCommunityInvalidId(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())

	resp := updateCommunityViaApi(t, app, "abc", token, []byte(`{"name":"Comunidade Id Invalido"}`))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "Invalid params" {
		t.Errorf("esperava message 'Invalid params', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "id must be a valid UUID v7" {
		t.Errorf("esperava cause do campo 'id', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateCommunityWithoutToken(t *testing.T) {
	app := setupApp()
	req := newUpdateCommunityRequest(uuidv7.New().String(), []byte(`{"name":"Comunidade Sem Token"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401, recebeu %d", resp.StatusCode)
	}
}

func TestUpdateCommunityTokenOfUnknownUser(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.unknowntoken@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Token Invalido")

	resp := updateCommunityViaApi(t, app, communityId, validTokenFor(t, uuidv7.New().String()), []byte(`{"name":"Comunidade Token Invalido"}`))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func TestUpdateCommunityIgnoresOwnerIdInBody(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.ownerid@ajuda.dev")
	other := createCommunityOwner(t, "other.update.ownerid@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update Owner Ignorado")

	body := []byte(`{"name":"Comunidade Owner Ignorado Renomeada","owner_id":"` + other.Id + `"}`)
	resp := updateCommunityViaApi(t, app, communityId, token, body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Name != "Comunidade Owner Ignorado Renomeada" {
		t.Errorf("esperava name 'Comunidade Owner Ignorado Renomeada', recebeu '%s'", updated.Name)
	}
	if updated.Owner == nil || updated.Owner.Id != owner.Id {
		t.Fatalf("owner_id do body deve ser ignorado: esperava owner '%s', recebeu %+v", owner.Id, updated.Owner)
	}

	persisted := getCommunityById(t, app, communityId, token)
	if persisted.Owner == nil || persisted.Owner.Id != owner.Id {
		t.Errorf("esperava owner '%s' persistido, recebeu %+v", owner.Id, persisted.Owner)
	}
}

func TestUpdateCommunityNoOpReturnsOk(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.update.noop@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Update No Op")

	body := []byte(`{"name":"Comunidade Update No Op","description":"comunidade de teste","address_id":"` + address.Id + `"}`)
	resp := updateCommunityViaApi(t, app, communityId, token, body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("no-op não pode devolver 404: esperava 200, recebeu %d", resp.StatusCode)
	}
	updated := decodeCommunityDto(t, resp)
	if updated.Id != communityId || updated.Name != "Comunidade Update No Op" {
		t.Errorf("esperava a comunidade '%s' inalterada, recebeu %+v", communityId, updated)
	}
}
