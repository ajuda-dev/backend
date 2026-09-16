package controller_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	communitydto "github.com/ajuda-dev/backend/src/controller/community/dto"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newCommunityRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/community/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateCommunitySuccess(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})

	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	address, a_err := addressRepository.CreateAddress(&addressdomain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if a_err != nil {
		t.Fatalf("failed to create address: %v", a_err)
	}
	body := []byte(`
	{
	"address_id": "` + address.Id + `",
	"owner_id": "` + user.Id + `",
	"name": "Dev Mode Community",
	"description": "comunidade de desenvolvedores de campos"
	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	var respDto communitydto.CommunityDto
	if err := json.Unmarshal(rawBody, &respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	var decodedBody map[string]interface{}
	if err := json.Unmarshal(rawBody, &decodedBody); err != nil {
		t.Fatalf("erro ao decodificar body como map: %v", err)
	}
	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
	if respDto.Name != "Dev Mode Community" {
		t.Errorf("esperava name 'Dev Mode Community', recebeu '%s'", respDto.Name)
	}
	if respDto.Description != "comunidade de desenvolvedores de campos" {
		t.Errorf("esperava a description enviada, recebeu '%s'", respDto.Description)
	}
	if respDto.Address == nil || respDto.Address.Id != address.Id {
		t.Fatalf("esperava address.id '%s' no 201, recebeu %+v", address.Id, respDto.Address)
	}
	if respDto.Address.City != "test_city" {
		t.Errorf("esperava address.city 'test_city' no 201, recebeu '%s'", respDto.Address.City)
	}
	if respDto.Owner == nil || respDto.Owner.Id != user.Id {
		t.Fatalf("esperava owner.id '%s' no 201, recebeu %+v", user.Id, respDto.Owner)
	}
	if respDto.Owner.Name != "teste" || respDto.Owner.Email != testEmail || respDto.Owner.Role != "USER" {
		t.Errorf("esperava owner completo no 201, recebeu %+v", respDto.Owner)
	}
	if respDto.Owner.Token != "" {
		t.Errorf("esperava owner.token vazio no 201, recebeu '%s'", respDto.Owner.Token)
	}
	if _, ok := decodedBody["owner_id"]; ok {
		t.Errorf("o 201 não deve expor owner_id (contrato novo), recebeu %v", decodedBody)
	}
	if _, ok := decodedBody["address_id"]; ok {
		t.Errorf("o 201 não deve expor address_id (contrato novo), recebeu %v", decodedBody)
	}
}

func TestCreateCommunityFail(t *testing.T) {
	t.Cleanup(cleanCommunityTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(`
	{	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400, recebeu %d", resp.StatusCode)
	}

	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "Invalid community data" {
		t.Errorf("esperava message 'Invalid user data', recebeu '%s'", respBody.Message)
	}
	verifyCodeError(t, respBody)
	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Name is not valid" {
		t.Errorf("esperava cause para o campo 'name', recebeu %+v", respBody.Causes)
	}
	causes = nil
	causes = getCauseByField("description", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Description is not valid" {
		t.Errorf("esperava cause para o campo 'description', recebeu %+v", respBody.Causes)
	}
	causes = nil
	causes = getCauseByField("addressId", respBody.Causes)
	if len(causes) == 0 || causes[0] != "AddressId is not valid" {
		t.Errorf("esperava cause para o campo 'addressId', recebeu %+v", respBody.Causes)
	}
	causes = nil
	causes = getCauseByField("ownerId", respBody.Causes)
	if len(causes) != 0 {
		t.Errorf("o owner agora vem do token; não esperava cause para o campo 'ownerId', recebeu %+v", respBody.Causes)
	}
}

func createCommunityUser(t *testing.T) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

func createCommunityOwner(t *testing.T, email string) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "owner",
		Email:    email,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create owner: %v", createErr)
	}
	return user
}

func createCommunityAddress(t *testing.T, city string) *addressdomain.AddressDomain {
	t.Helper()
	address, aErr := addressRepository.CreateAddress(&addressdomain.AddressDomain{
		City:    city,
		State:   "sp",
		Street:  "test_street",
		ZipCode: "test_zip_code_" + city,
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}
	return address
}

func registerCommunityViaApi(t *testing.T, app *fiber.App, token string, addressId string, name string) string {
	t.Helper()
	body := []byte(`{
	"address_id": "` + addressId + `",
	"name": "` + name + `",
	"description": "comunidade de teste"
	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 ao criar comunidade '%s', recebeu %d", name, resp.StatusCode)
	}
	var created communitydto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return created.Id
}

func registerCommunityForOwner(t *testing.T, app *fiber.App, token string, addressId string, name string) string {
	t.Helper()
	body := []byte(`{
	"address_id": "` + addressId + `",
	"name": "` + name + `",
	"description": "comunidade de teste"
	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 ao criar comunidade '%s', recebeu %d", name, resp.StatusCode)
	}
	var created communitydto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return created.Id
}

func listCommunities(t *testing.T, app *fiber.App, token string, query string) communitydto.PageableCommunityDto {
	t.Helper()
	req := httptest.NewRequest("GET", "/v1/community?"+query, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page communitydto.PageableCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func TestListCommunitiesByCity(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	spCommunity := registerCommunityViaApi(t, app, token, addressSaoPaulo.Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	page := listCommunities(t, app, token, "city=sao%20paulo")
	if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
		t.Errorf("esperava somente a comunidade de sao paulo, recebeu %+v", page.Data)
	}
	if len(page.Data) == 1 && (page.Data[0].Address == nil || page.Data[0].Address.City != "sao paulo") {
		t.Errorf("esperava address.city 'sao paulo' no item, recebeu %+v", page.Data[0].Address)
	}
}

func TestListCommunitiesCaseInsensitive(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	spCommunity := registerCommunityViaApi(t, app, token, addressSaoPaulo.Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	page := listCommunities(t, app, token, "city=SAO%20PAULO")
	if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
		t.Errorf("esperava somente a comunidade de sao paulo (case-insensitive), recebeu %+v", page.Data)
	}
}

func TestListCommunitiesNoMatch(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	registerCommunityViaApi(t, app, token, addressSaoPaulo.Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	page := listCommunities(t, app, token, "city=nao%20existe")
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListCommunitiesPagination(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Um")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Dois")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Tres")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	pageOne := listCommunities(t, app, token, "city=sao%20paulo&limit=2")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}

	pageTwo := listCommunities(t, app, token, "city=sao%20paulo&limit=2&page=2")
	if len(pageTwo.Data) != 1 || pageTwo.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
}

func TestListCommunitiesSubstring(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	spCommunity := registerCommunityViaApi(t, app, token, addressSaoPaulo.Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	queries := []string{"city=paulo", "city=sao", "city=o%20pa", "city=PaUlO"}
	for _, query := range queries {
		page := listCommunities(t, app, token, query)
		if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
			t.Errorf("query %q: esperava somente a comunidade de sao paulo, recebeu %+v", query, page.Data)
		}
	}
}

func TestListCommunitiesSubstringMatchesMultipleCities(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)

	saoPauloCommunity := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")
	pauloAfonsoCommunity := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "paulo afonso").Id, "Comunidade Paulo Afonso")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	page := listCommunities(t, app, token, "city=paulo")
	if len(page.Data) != 2 {
		t.Fatalf("esperava 2 comunidades contendo 'paulo', recebeu %d: %+v", len(page.Data), page.Data)
	}
	found := map[string]bool{}
	for _, item := range page.Data {
		found[item.Id] = true
	}
	if !found[saoPauloCommunity] || !found[pauloAfonsoCommunity] {
		t.Errorf("esperava as comunidades %s e %s, recebeu %+v", saoPauloCommunity, pauloAfonsoCommunity, page.Data)
	}
}

func TestListCommunitiesWildcardEscape(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")

	wildcardPage := listCommunities(t, app, token, "city=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listCommunities(t, app, token, "city=sao_paulo")
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'sao_paulo' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListCommunitiesWhitespaceOnly(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	page := listCommunities(t, app, token, "city=%20%20")
	if len(page.Data) != 2 || page.HasNext {
		t.Errorf("esperava a lista completa (2 itens) sem filtro, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}

	trimmedPage := listCommunities(t, app, token, "city=%20paulo")
	if len(trimmedPage.Data) != 1 || trimmedPage.Data[0].Address == nil || trimmedPage.Data[0].Address.City != "sao paulo" {
		t.Errorf("esperava 1 resultado para ' paulo' (com trim), recebeu %+v", trimmedPage.Data)
	}
}

func TestListCommunitiesSubstringPagination(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Um")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Dois")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Tres")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	pageOne := listCommunities(t, app, token, "city=sao&limit=2")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}

	pageTwo := listCommunities(t, app, token, "city=sao&limit=2&page=2")
	if len(pageTwo.Data) != 1 || pageTwo.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
}

func TestListCommunitiesByCityAccentInsensitive(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	community := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "são paulo").Id, "Comunidade São Paulo")

	queries := []string{"city=sao", "city=SAO", "city=paulo", "city=são"}
	for _, query := range queries {
		page := listCommunities(t, app, token, query)
		if len(page.Data) != 1 || page.Data[0].Id != community {
			t.Errorf("query %q: esperava a comunidade '%s', recebeu %+v", query, community, page.Data)
		}
	}

	noMatchPage := listCommunities(t, app, token, "city=nao%20existe")
	if len(noMatchPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'nao existe', recebeu %+v", noMatchPage.Data)
	}
}

func TestListCommunitiesByCityAccentInsensitiveReverse(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	community := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo Sem Acento")

	page := listCommunities(t, app, token, "city=são%20paulo")
	if len(page.Data) != 1 || page.Data[0].Id != community {
		t.Errorf("esperava a comunidade '%s' para termo acentuado sobre base sem acento, recebeu %+v", community, page.Data)
	}
}

func registerCommunityReturningBody(t *testing.T, app *fiber.App, token string, addressId string, name string) communitydto.CommunityDto {
	t.Helper()
	body := []byte(`{
	"address_id": "` + addressId + `",
	"name": "` + name + `",
	"description": "comunidade de teste"
	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 ao criar comunidade '%s', recebeu %d", name, resp.StatusCode)
	}
	var created communitydto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return created
}

func getCommunityByIdRequest(t *testing.T, app *fiber.App, token string, id string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("GET", "/v1/community/"+id, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func TestGetCommunityById(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")
	created := registerCommunityReturningBody(t, app, token, address.Id, "Comunidade Detalhe")

	resp := getCommunityByIdRequest(t, app, token, created.Id)
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET por id, recebeu %d", resp.StatusCode)
	}
	var detail communitydto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if !reflect.DeepEqual(created, detail) {
		t.Errorf("esperava o mesmo corpo do 201 no GET\n201:  %+v\nGET:  %+v", created, detail)
	}
}

func TestGetCommunityByIdInvalidId(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())

	for _, id := range []string{"abc", "123", "not-a-uuid"} {
		resp := getCommunityByIdRequest(t, app, token, id)
		if resp.StatusCode != fiber.StatusBadRequest {
			resp.Body.Close()
			t.Errorf("id %q: esperava 400, recebeu %d", id, resp.StatusCode)
			continue
		}
		var respBody rest_err.RestErr
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			resp.Body.Close()
			t.Fatalf("id %q: erro ao decodificar body: %v", id, err)
		}
		resp.Body.Close()
		causes := getCauseByField("id", respBody.Causes)
		if len(causes) == 0 || causes[0] != "id must be a valid UUID v7" {
			t.Errorf("id %q: esperava cause do campo 'id', recebeu %+v", id, respBody.Causes)
		}
	}
}

func TestGetCommunityByIdNotFound(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())

	resp := getCommunityByIdRequest(t, app, token, uuidv7.New().String())
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para id inexistente, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "community not found" {
		t.Errorf("esperava message 'community not found', recebeu '%s'", respBody.Message)
	}
}

func TestGetCommunityByIdWithoutToken(t *testing.T) {
	app := setupApp()

	req := httptest.NewRequest("GET", "/v1/community/"+uuidv7.New().String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
}

func TestGetCommunityByIdSoftDeleted(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")
	created := registerCommunityReturningBody(t, app, token, address.Id, "Comunidade Removida")

	deleteReq := httptest.NewRequest("DELETE", "/v1/community/"+created.Id, nil)
	deleteResp, err := doAuthedRequest(app, deleteReq, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	deleteResp.Body.Close()
	if deleteResp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 no delete, recebeu %d", deleteResp.StatusCode)
	}

	resp := getCommunityByIdRequest(t, app, token, created.Id)
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para comunidade arquivada, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "community not found" {
		t.Errorf("esperava message 'community not found', recebeu '%s'", respBody.Message)
	}
}

func TestGetCommunityByIdOtherOwner(t *testing.T) {
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	owner := createCommunityUser(t)
	address := createCommunityAddress(t, "sao paulo")
	created := registerCommunityReturningBody(t, app, validTokenFor(t, owner.Id), address.Id, "Comunidade de Outro Dono")

	other := createUserWithRole(t, "outro.dono.comunidade@ajuda.dev", userdomain.UserRoleUser)
	resp := getCommunityByIdRequest(t, app, validTokenFor(t, other.Id), created.Id)
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("leitura não é restrita por owner: esperava 200, recebeu %d", resp.StatusCode)
	}
	var detail communitydto.CommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if detail.Id != created.Id || detail.Owner == nil || detail.Owner.Id != owner.Id {
		t.Errorf("esperava a comunidade %s com owner %s, recebeu %+v", created.Id, owner.Id, detail)
	}
}

func TestListCommunitiesByOwnerId(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.ownerid@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.ownerid@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)
	address := createCommunityAddress(t, "sao paulo")

	communityA1 := registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Owner A Um")
	communityA2 := registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Owner A Dois")
	registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade Owner B Um")
	registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade Owner B Dois")

	page := listCommunities(t, app, tokenA, "owner_id="+ownerA.Id)
	if len(page.Data) != 2 || page.HasNext {
		t.Fatalf("esperava 2 comunidades do owner A, recebeu %d itens, has_next=%v: %+v", len(page.Data), page.HasNext, page.Data)
	}
	found := map[string]bool{}
	for _, item := range page.Data {
		found[item.Id] = true
		if item.Owner == nil || item.Owner.Id != ownerA.Id {
			t.Errorf("esperava owner.id '%s' no item, recebeu %+v", ownerA.Id, item.Owner)
		}
	}
	if !found[communityA1] || !found[communityA2] {
		t.Errorf("esperava as comunidades %s e %s, recebeu %+v", communityA1, communityA2, page.Data)
	}
}

func TestListCommunitiesByOwnerIdAndCity(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.ownercity@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.ownercity@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)

	communityA := registerCommunityForOwner(t, app, tokenA, createCommunityAddress(t, "sao paulo").Id, "Comunidade Owner A Sao Paulo")
	registerCommunityForOwner(t, app, tokenA, createCommunityAddress(t, "campinas").Id, "Comunidade Owner A Campinas")
	registerCommunityForOwner(t, app, tokenB, createCommunityAddress(t, "sao paulo").Id, "Comunidade Owner B Sao Paulo")

	page := listCommunities(t, app, tokenA, "city=sao&owner_id="+ownerA.Id)
	if len(page.Data) != 1 || page.Data[0].Id != communityA {
		t.Errorf("esperava somente a comunidade do owner A em sao paulo, recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByOwnerIdInvalidUuid(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())

	for _, ownerId := range []string{"abc", "123"} {
		req := httptest.NewRequest("GET", "/v1/community?owner_id="+ownerId, nil)
		resp, err := doAuthedRequest(app, req, token)
		if err != nil {
			t.Fatalf("owner_id %q: erro ao executar requisição: %v", ownerId, err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			resp.Body.Close()
			t.Errorf("owner_id %q: esperava 400, recebeu %d", ownerId, resp.StatusCode)
			continue
		}
		var respBody rest_err.RestErr
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			resp.Body.Close()
			t.Fatalf("owner_id %q: erro ao decodificar body: %v", ownerId, err)
		}
		resp.Body.Close()
		if respBody.Message != "Invalid query params" {
			t.Errorf("owner_id %q: esperava message 'Invalid query params', recebeu '%s'", ownerId, respBody.Message)
		}
		causes := getCauseByField("owner_id", respBody.Causes)
		if len(causes) == 0 || causes[0] != "owner_id must be a valid UUID v7" {
			t.Errorf("owner_id %q: esperava cause do campo 'owner_id', recebeu %+v", ownerId, respBody.Causes)
		}
	}
}

func TestListCommunitiesByOwnerIdNoMatch(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	owner := createCommunityOwner(t, "owner.nomatch@ajuda.dev")
	token := validTokenFor(t, owner.Id)
	registerCommunityForOwner(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Owner No Match")

	page := listCommunities(t, app, token, "owner_id="+uuidv7.New().String())
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListCommunitiesByOwnerIdPagination(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.ownerpage@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.ownerpage@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)
	address := createCommunityAddress(t, "sao paulo")

	registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Owner A Pagina Um")
	registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Owner A Pagina Dois")
	registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Owner A Pagina Tres")
	registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade Owner B Pagina Um")

	pageOne := listCommunities(t, app, tokenA, "owner_id="+ownerA.Id+"&limit=2")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}

	pageTwo := listCommunities(t, app, tokenA, "owner_id="+ownerA.Id+"&limit=2&page=2")
	if len(pageTwo.Data) != 1 || pageTwo.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
}

func TestListCommunitiesIgnoresAddressIdQueryParam(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressOne := createCommunityAddress(t, "sao paulo")
	addressTwo := createCommunityAddress(t, "campinas")

	registerCommunityViaApi(t, app, token, addressOne.Id, "Comunidade Ignora Address Um")
	registerCommunityViaApi(t, app, token, addressTwo.Id, "Comunidade Ignora Address Dois")

	page := listCommunities(t, app, token, "address_id="+addressOne.Id)
	if len(page.Data) != 2 || page.HasNext {
		t.Errorf("address_id deve ser ignorado: esperava as 2 comunidades, recebeu %d itens, has_next=%v: %+v", len(page.Data), page.HasNext, page.Data)
	}
}

func TestListCommunitiesByNameSubstring(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	devSp := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev SP")
	devRj := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev RJ")
	registerCommunityViaApi(t, app, token, address.Id, "Comunidade Design SP")

	page := listCommunities(t, app, token, "name=dev")
	if len(page.Data) != 2 {
		t.Fatalf("esperava 2 comunidades contendo 'dev', recebeu %d: %+v", len(page.Data), page.Data)
	}
	found := map[string]bool{}
	for _, item := range page.Data {
		found[item.Id] = true
	}
	if !found[devSp] || !found[devRj] {
		t.Errorf("esperava as comunidades %s e %s, recebeu %+v", devSp, devRj, page.Data)
	}
}

func TestListCommunitiesByNameCaseInsensitive(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	community := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev SP")

	queries := []string{"name=DEV", "name=dev%20sp", "name=Comunidade"}
	for _, query := range queries {
		page := listCommunities(t, app, token, query)
		if len(page.Data) != 1 || page.Data[0].Id != community {
			t.Errorf("query %q: esperava somente a comunidade '%s', recebeu %+v", query, community, page.Data)
		}
	}
}

func TestListCommunitiesByNameAndOwnerId(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.ownername@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.ownername@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)
	address := createCommunityAddress(t, "sao paulo")

	communityA := registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Dev SP")
	registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade Design")
	registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade Dev RJ")

	page := listCommunities(t, app, tokenA, "owner_id="+ownerA.Id+"&name=dev")
	if len(page.Data) != 1 || page.Data[0].Id != communityA {
		t.Errorf("esperava somente a comunidade de dev do owner A, recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByNameEscape(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev SP")

	wildcardPage := listCommunities(t, app, token, "name=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listCommunities(t, app, token, "name=dev_sp")
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'dev_sp' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListCommunitiesByNameWhitespace(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	community := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev SP")
	registerCommunityViaApi(t, app, token, address.Id, "Comunidade Design")

	whitespacePage := listCommunities(t, app, token, "name=%20%20")
	if len(whitespacePage.Data) != 2 || whitespacePage.HasNext {
		t.Errorf("esperava a lista completa (2 itens) sem filtro, recebeu %d itens, has_next=%v", len(whitespacePage.Data), whitespacePage.HasNext)
	}

	trimmedPage := listCommunities(t, app, token, "name=%20dev")
	if len(trimmedPage.Data) != 1 || trimmedPage.Data[0].Id != community {
		t.Errorf("esperava 1 resultado para ' dev' (com trim), recebeu %+v", trimmedPage.Data)
	}
}

func TestListCommunitiesByNameAccentInsensitive(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	community := registerCommunityViaApi(t, app, token, address.Id, "Comunidade São Paulo")

	queries := []string{"name=sao", "name=SÃO", "name=paulo"}
	for _, query := range queries {
		page := listCommunities(t, app, token, query)
		if len(page.Data) != 1 || page.Data[0].Id != community {
			t.Errorf("query %q: esperava a comunidade '%s', recebeu %+v", query, community, page.Data)
		}
	}
}

func TestListCommunitiesByNameAccentInsensitiveReverse(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	community := registerCommunityViaApi(t, app, token, address.Id, "Comunidade Acai Belem")

	page := listCommunities(t, app, token, "name=açaí")
	if len(page.Data) != 1 || page.Data[0].Id != community {
		t.Errorf("esperava a comunidade '%s' para termo acentuado sobre nome sem acento, recebeu %+v", community, page.Data)
	}
}

func TestListCommunitiesByNameAccentWithOwnerId(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.owneraccent@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.owneraccent@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)
	address := createCommunityAddress(t, "sao paulo")

	communityA := registerCommunityForOwner(t, app, tokenA, address.Id, "Comunidade São Paulo")
	registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade São Paulo Belem")

	page := listCommunities(t, app, tokenA, "owner_id="+ownerA.Id+"&name=sao")
	if len(page.Data) != 1 || page.Data[0].Id != communityA {
		t.Errorf("esperava somente a comunidade de A, recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByNameAccentEscape(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	address := createCommunityAddress(t, "sao paulo")

	registerCommunityViaApi(t, app, token, address.Id, "Comunidade Dev SP")

	wildcardPage := listCommunities(t, app, token, "name=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listCommunities(t, app, token, "name=dev_sp")
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'dev_sp' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListCommunitiesOwnerIdOfOtherUserIsReadable(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	ownerA := createCommunityOwner(t, "owner.a.ownerread@ajuda.dev")
	ownerB := createCommunityOwner(t, "owner.b.ownerread@ajuda.dev")
	tokenA := validTokenFor(t, ownerA.Id)
	tokenB := validTokenFor(t, ownerB.Id)
	address := createCommunityAddress(t, "sao paulo")

	communityB := registerCommunityForOwner(t, app, tokenB, address.Id, "Comunidade Owner B Leitura")

	page := listCommunities(t, app, tokenA, "owner_id="+ownerB.Id)
	if len(page.Data) != 1 || page.Data[0].Id != communityB {
		t.Errorf("leitura não é restrita por owner: esperava a comunidade %s, recebeu %+v", communityB, page.Data)
	}
}
