package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
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
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})


	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	address, a_err := addressRepository.CreateAddress(&domain.AddressDomain{
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

	var respDto dto.RegisterCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
}


func TestCreateCommunityFail(t *testing.T){
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

func createCommunityUser(t *testing.T) *domain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

func createCommunityAddress(t *testing.T, city string) *domain.AddressDomain {
	t.Helper()
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
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
	var created dto.RegisterCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return created.Id
}

func listCommunitiesByCity(t *testing.T, app *fiber.App, token string, query string) dto.PageableCommunityDto {
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
	var page dto.PageableCommunityDto
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

	page := listCommunitiesByCity(t, app, token, "city=sao%20paulo")
	if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
		t.Errorf("esperava somente a comunidade de sao paulo, recebeu %+v", page.Data)
	}
	if len(page.Data) == 1 && (page.Data[0].Address == nil || page.Data[0].Address.City != "sao paulo") {
		t.Errorf("esperava address.city 'sao paulo' no item, recebeu %+v", page.Data[0].Address)
	}
}

func TestListCommunitiesByCityCaseInsensitive(t *testing.T) {
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

	page := listCommunitiesByCity(t, app, token, "city=SAO%20PAULO")
	if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
		t.Errorf("esperava somente a comunidade de sao paulo (case-insensitive), recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByCityAndAddressId(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo1 := createCommunityAddress(t, "sao paulo")
	addressSaoPaulo2 := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	firstCommunity := registerCommunityViaApi(t, app, token, addressSaoPaulo1.Id, "Comunidade Paulista Primeira")
	registerCommunityViaApi(t, app, token, addressSaoPaulo2.Id, "Comunidade Paulista Segunda")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	page := listCommunitiesByCity(t, app, token, "city=sao%20paulo&address_id="+addressSaoPaulo1.Id)
	if len(page.Data) != 1 || page.Data[0].Id != firstCommunity {
		t.Errorf("esperava somente a comunidade do endereço 1 em sao paulo, recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByCityNoMatch(t *testing.T) {
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

	page := listCommunitiesByCity(t, app, token, "city=nao%20existe")
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListCommunitiesByCityPagination(t *testing.T) {
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

	pageOne := listCommunitiesByCity(t, app, token, "city=sao%20paulo&limit=2")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}

	pageTwo := listCommunitiesByCity(t, app, token, "city=sao%20paulo&limit=2&page=2")
	if len(pageTwo.Data) != 1 || pageTwo.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
}

func TestListCommunitiesByCitySubstring(t *testing.T) {
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
		page := listCommunitiesByCity(t, app, token, query)
		if len(page.Data) != 1 || page.Data[0].Id != spCommunity {
			t.Errorf("query %q: esperava somente a comunidade de sao paulo, recebeu %+v", query, page.Data)
		}
	}
}

func TestListCommunitiesByCitySubstringMatchesMultipleCities(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)

	saoPauloCommunity := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")
	pauloAfonsoCommunity := registerCommunityViaApi(t, app, token, createCommunityAddress(t, "paulo afonso").Id, "Comunidade Paulo Afonso")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	page := listCommunitiesByCity(t, app, token, "city=paulo")
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

func TestListCommunitiesByCitySubstringAndAddressId(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	addressSaoPaulo1 := createCommunityAddress(t, "sao paulo")
	addressSaoPaulo2 := createCommunityAddress(t, "sao paulo")
	addressCampinas := createCommunityAddress(t, "campinas")

	firstCommunity := registerCommunityViaApi(t, app, token, addressSaoPaulo1.Id, "Comunidade Paulista Primeira")
	registerCommunityViaApi(t, app, token, addressSaoPaulo2.Id, "Comunidade Paulista Segunda")
	registerCommunityViaApi(t, app, token, addressCampinas.Id, "Comunidade Campinas")

	page := listCommunitiesByCity(t, app, token, "city=sao&address_id="+addressSaoPaulo1.Id)
	if len(page.Data) != 1 || page.Data[0].Id != firstCommunity {
		t.Errorf("esperava somente a comunidade do endereço 1 em sao paulo, recebeu %+v", page.Data)
	}
}

func TestListCommunitiesByCityWildcardEscape(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")

	wildcardPage := listCommunitiesByCity(t, app, token, "city=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listCommunitiesByCity(t, app, token, "city=sao_paulo")
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'sao_paulo' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListCommunitiesByCityWhitespaceOnly(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "sao paulo").Id, "Comunidade Sao Paulo")
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "campinas").Id, "Comunidade Campinas")

	page := listCommunitiesByCity(t, app, token, "city=%20%20")
	if len(page.Data) != 2 || page.HasNext {
		t.Errorf("esperava a lista completa (2 itens) sem filtro, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}

	trimmedPage := listCommunitiesByCity(t, app, token, "city=%20paulo")
	if len(trimmedPage.Data) != 1 || trimmedPage.Data[0].Address == nil || trimmedPage.Data[0].Address.City != "sao paulo" {
		t.Errorf("esperava 1 resultado para ' paulo' (com trim), recebeu %+v", trimmedPage.Data)
	}
}

func TestListCommunitiesByCitySubstringPagination(t *testing.T) {
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

	pageOne := listCommunitiesByCity(t, app, token, "city=sao&limit=2")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}

	pageTwo := listCommunitiesByCity(t, app, token, "city=sao&limit=2&page=2")
	if len(pageTwo.Data) != 1 || pageTwo.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
}

func TestListCommunitiesByCityAccentNotNormalized(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	app := setupApp()
	user := createCommunityUser(t)
	token := validTokenFor(t, user.Id)
	registerCommunityViaApi(t, app, token, createCommunityAddress(t, "são paulo").Id, "Comunidade São Paulo")

	page := listCommunitiesByCity(t, app, token, "city=sao")
	if len(page.Data) != 0 {
		t.Errorf("acentos não são normalizados: esperava 0 resultados para 'sao', recebeu %+v", page.Data)
	}
}
