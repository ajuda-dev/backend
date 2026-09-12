package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func TestRegisterAddressSuccess(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(
		`
	 	{
  		"zip_code": "test_zip_code",
  		"number": "123",
  		"complement": "ap 5"
		}
	 `)
	req := newAddressRegisterRequest(body)

	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}

	var respDto dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
	if respDto.Number != "123" {
		t.Errorf("esperava number '123', recebeu '%s'", respDto.Number)
	}
	if respDto.Complement != "ap 5" {
		t.Errorf("esperava complement 'ap 5', recebeu '%s'", respDto.Complement)
	}
	if respDto.Street != "mock street" {
		t.Errorf("esperava street 'mock street' vinda do ViaCEP via merge, recebeu '%s'", respDto.Street)
	}
	if respDto.ZipCode != "12345-678" {
		t.Errorf("esperava zip_code '12345-678' vindo do ViaCEP via merge, recebeu '%s'", respDto.ZipCode)
	}

}

func newAddressRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/address/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestRegisterAddressBadRequest(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(
		`
	 	{
			"city": "test_city",
			"state": "test_state",
			"street": "test_street",
			"number": "123"
		}
	 `)
	req := newAddressRegisterRequest(body)

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
	if respBody.Message != "Invalid address data" {
		t.Errorf("esperava message 'Invalid address data', recebeu '%s'", respBody.Message)
	}

}

func TestRegisterAddressAlreadyExist(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		ZipCode: "test_zip_code",
	})
	body := []byte(
		`
	 	{
			"zip_code": "test_zip_code"
		}
	 `)
	req := newAddressRegisterRequest(body)

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
	if respBody.Message != "Invalid address data" {
		t.Errorf("esperava message 'Invalid user data', recebeu '%s'", respBody.Message)
	}
	verifyCodeError(t, respBody)
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "address" || respBody.Causes[0].Message != "Address already exists" {
		t.Errorf("esperava cause para address já existente, recebeu %+v", respBody.Causes)
	}

}

func TestRegisterAddressDuplicateFullAddress(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(
		`
	 	{
			"zip_code": "test_zip_code",
			"number": "321",
			"complement": "sala 3"
		}
	 `)
	req := newAddressRegisterRequest(body)

	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 na primeira requisição, recebeu %d", resp.StatusCode)
	}

	req = newAddressRegisterRequest(body)
	resp, err = doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na repetição do mesmo endereço, recebeu %d", resp.StatusCode)
	}

	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "Invalid address data" {
		t.Errorf("esperava message 'Invalid address data', recebeu '%s'", respBody.Message)
	}
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "address" || respBody.Causes[0].Message != "Address already exists" {
		t.Errorf("esperava cause para address já existente, recebeu %+v", respBody.Causes)
	}
}

func TestRegisterAddressSameCepDifferentNumber(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())

	body := []byte(
		`
	 	{
			"zip_code": "test_zip_code",
			"number": "100"
		}
	 `)
	req := newAddressRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 para number 100, recebeu %d", resp.StatusCode)
	}
	var firstDto dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&firstDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	body = []byte(
		`
	 	{
			"zip_code": "test_zip_code",
			"number": "200"
		}
	 `)
	req = newAddressRegisterRequest(body)
	resp, err = doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 para number 200, recebeu %d", resp.StatusCode)
	}
	var secondDto dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&secondDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	if firstDto.Id == secondDto.Id {
		t.Errorf("esperava ids diferentes para números diferentes no mesmo CEP, recebeu '%s'", firstDto.Id)
	}
}

func TestRegisterAddressPersistenceRoundTrip(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(
		`
	 	{
			"zip_code": "test_zip_code",
			"number": "  456  ",
			"complement": "SALA 5"
		}
	 `)
	req := newAddressRegisterRequest(body)

	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}

	var respDto dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	persisted, getErr := addressRepository.GetAddressById(respDto.Id)
	if getErr != nil {
		t.Fatalf("erro ao buscar endereço persistido: %v", getErr)
	}
	if persisted.Street != "mock street" {
		t.Errorf("esperava street 'mock street' normalizada em minúsculo, recebeu '%s'", persisted.Street)
	}
	if persisted.Number != "456" {
		t.Errorf("esperava number '456' com trim, recebeu '%s'", persisted.Number)
	}
	if persisted.Complement != "sala 5" {
		t.Errorf("esperava complement 'sala 5' normalizada em minúsculo, recebeu '%s'", persisted.Complement)
	}
}

func createAddressForSearch(t *testing.T, city string, state string, zipCode string, street string, number string) *domain.AddressDomain {
	t.Helper()
	t.Cleanup(cleanAddressesTable)
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    city,
		State:   state,
		Street:  street,
		ZipCode: zipCode,
		Number:  number,
	})
	if aErr != nil {
		t.Fatalf("failed to create address for search: %v", aErr)
	}
	return address
}

func listAddresses(t *testing.T, app *fiber.App, token string, query string) dto.PageableAddressDto {
	t.Helper()
	req := httptest.NewRequest("GET", "/v1/address?"+query, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page dto.PageableAddressDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func TestListAddressesWithoutFilter(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")
	createAddressForSearch(t, "campinas", "SP", "13010-100", "rua treze de maio", "200")

	page := listAddresses(t, app, token, "")
	if len(page.Data) != 2 || page.HasNext {
		t.Errorf("esperava 2 itens com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesHasNext(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")
	createAddressForSearch(t, "sao paulo", "SP", "01310-101", "avenida paulista", "1001")
	createAddressForSearch(t, "sao paulo", "SP", "01310-102", "avenida paulista", "1002")

	page := listAddresses(t, app, token, "limit=2")
	if len(page.Data) != 2 || !page.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesByCityPartial(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	saoPaulo := createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")
	createAddressForSearch(t, "campinas", "SP", "13010-100", "rua treze de maio", "200")

	page := listAddresses(t, app, token, "city=sao")
	if len(page.Data) != 1 || page.Data[0].Id != saoPaulo.Id {
		t.Errorf("esperava somente o endereço de sao paulo, recebeu %+v", page.Data)
	}
}

func TestListAddressesByCityCaseInsensitive(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	saoPaulo := createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "city=SAO%20PAULO")
	if len(page.Data) != 1 || page.Data[0].Id != saoPaulo.Id {
		t.Errorf("esperava o endereço de sao paulo (case-insensitive), recebeu %+v", page.Data)
	}
}

func TestListAddressesByCityAccentInsensitive(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	belem := createAddressForSearch(t, "belem", "PA", "66010-000", "avenida presidente vargas", "100")

	page := listAddresses(t, app, token, "city=bel%C3%A9m")
	if len(page.Data) != 1 || page.Data[0].Id != belem.Id {
		t.Errorf("esperava o endereço 'belem' buscando por 'belém', recebeu %+v", page.Data)
	}
}

func TestListAddressesByCityAccentInsensitiveInverse(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	belem := createAddressForSearch(t, "belém", "PA", "66010-000", "avenida presidente vargas", "100")

	page := listAddresses(t, app, token, "city=belem")
	if len(page.Data) != 1 || page.Data[0].Id != belem.Id {
		t.Errorf("esperava o endereço 'belém' buscando por 'belem', recebeu %+v", page.Data)
	}
}

func TestListAddressesByCityNoMatch(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "city=zzz")
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesByCityWildcardEscape(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	wildcardPage := listAddresses(t, app, token, "city=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listAddresses(t, app, token, "city=dev_sp")
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'dev_sp' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListAddressesByState(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	sp := createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")
	createAddressForSearch(t, "rio de janeiro", "RJ", "20040-020", "avenida rio branco", "1")

	page := listAddresses(t, app, token, "state=SP")
	if len(page.Data) != 1 || page.Data[0].Id != sp.Id {
		t.Errorf("esperava somente o endereço de SP, recebeu %+v", page.Data)
	}

	lowerPage := listAddresses(t, app, token, "state=sp")
	if len(lowerPage.Data) != 1 || lowerPage.Data[0].Id != sp.Id {
		t.Errorf("esperava o mesmo resultado para 'sp' (normalização para maiúsculo), recebeu %+v", lowerPage.Data)
	}
}

func TestListAddressesByStateFromApi(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(`{"zip_code": "test_zip_code", "number": "123"}`)
	req := newAddressRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d", resp.StatusCode)
	}
	var created dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	page := listAddresses(t, app, token, "state=MOCK%20STATE")
	if len(page.Data) != 1 || page.Data[0].Id != created.Id {
		t.Errorf("esperava o endereço criado via API com state 'Mock State', recebeu %+v", page.Data)
	}
}

func TestListAddressesByStateNoMatch(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "state=XYZ")
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia com has_next false, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesByZipCodeWithAndWithoutHyphen(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	address := createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	withHyphen := listAddresses(t, app, token, "zip_code=01310-100")
	if len(withHyphen.Data) != 1 || withHyphen.Data[0].Id != address.Id {
		t.Errorf("esperava o endereço buscando por '01310-100', recebeu %+v", withHyphen.Data)
	}

	withoutHyphen := listAddresses(t, app, token, "zip_code=01310100")
	if len(withoutHyphen.Data) != 1 || withoutHyphen.Data[0].Id != address.Id {
		t.Errorf("esperava o mesmo resultado buscando por '01310100', recebeu %+v", withoutHyphen.Data)
	}
}

func TestListAddressesByZipCodeStoredWithoutHyphen(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	address := createAddressForSearch(t, "sao paulo", "SP", "87654321", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "zip_code=87654-321")
	if len(page.Data) != 1 || page.Data[0].Id != address.Id {
		t.Errorf("esperava o endereço gravado sem hífen buscando por '87654-321', recebeu %+v", page.Data)
	}
}

func TestListAddressesByZipCodeWithoutDigits(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	req := httptest.NewRequest("GET", "/v1/address?zip_code=abc", nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "Invalid query params" {
		t.Errorf("esperava message 'Invalid query params', recebeu '%s'", respBody.Message)
	}
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "zip_code" || respBody.Causes[0].Message != "zip_code must contain digits" {
		t.Errorf("esperava cause para zip_code sem dígitos, recebeu %+v", respBody.Causes)
	}
}

func TestListAddressesByZipCodeIncomplete(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "zip_code=123")
	if len(page.Data) != 0 || page.HasNext {
		t.Errorf("esperava lista vazia (igualdade, não prefixo), recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesCombinedFilters(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	body := []byte(`{"zip_code": "test_zip_code", "number": "123"}`)
	req := newAddressRegisterRequest(body)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d", resp.StatusCode)
	}
	var created dto.AddressDto
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}

	page := listAddresses(t, app, token, "city=mock&state=MOCK%20STATE&zip_code=12345-678")
	if len(page.Data) != 1 || page.Data[0].Id != created.Id {
		t.Errorf("esperava o endereço com os 3 filtros combinados, recebeu %+v", page.Data)
	}

	noMatch := listAddresses(t, app, token, "city=mock&state=MOCK%20STATE&zip_code=99999-999")
	if len(noMatch.Data) != 0 || noMatch.HasNext {
		t.Errorf("esperava lista vazia variando o zip_code, recebeu %d itens, has_next=%v", len(noMatch.Data), noMatch.HasNext)
	}
}

func TestListAddressesSoftDeleted(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	address := createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")
	createAddressForSearch(t, "campinas", "SP", "13010-100", "rua treze de maio", "200")

	if err := db.Exec("UPDATE addresses SET deleted_at = NOW() WHERE id = ?", address.Id).Error; err != nil {
		t.Fatalf("erro ao arquivar endereço: %v", err)
	}

	page := listAddresses(t, app, token, "")
	if len(page.Data) != 1 || page.Data[0].Id == address.Id {
		t.Errorf("esperava somente o endereço não deletado, recebeu %+v", page.Data)
	}

	byZip := listAddresses(t, app, token, "zip_code=01310-100")
	if len(byZip.Data) != 0 {
		t.Errorf("esperava lista vazia para o CEP do endereço deletado, recebeu %+v", byZip.Data)
	}
}

func TestListAddressesPaginationWithFilter(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	for i := 0; i < 5; i++ {
		createAddressForSearch(t, "sao paulo", "SP", "01310-10"+strconv.Itoa(i), "avenida paulista", strconv.Itoa(1000+i))
	}

	pageOne := listAddresses(t, app, token, "city=sao&limit=2&page=1")
	if len(pageOne.Data) != 2 || !pageOne.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageOne.Data), pageOne.HasNext)
	}
	pageTwo := listAddresses(t, app, token, "city=sao&limit=2&page=2")
	if len(pageTwo.Data) != 2 || !pageTwo.HasNext {
		t.Errorf("esperava 2 itens com has_next true, recebeu %d itens, has_next=%v", len(pageTwo.Data), pageTwo.HasNext)
	}
	pageThree := listAddresses(t, app, token, "city=sao&limit=2&page=3")
	if len(pageThree.Data) != 1 || pageThree.HasNext {
		t.Errorf("esperava 1 item com has_next false, recebeu %d itens, has_next=%v", len(pageThree.Data), pageThree.HasNext)
	}

	seen := map[string]bool{}
	for _, page := range []dto.PageableAddressDto{pageOne, pageTwo, pageThree} {
		for _, item := range page.Data {
			if seen[item.Id] {
				t.Errorf("id '%s' repetido entre páginas (ordem não determinística)", item.Id)
			}
			seen[item.Id] = true
		}
	}
	if len(seen) != 5 {
		t.Errorf("esperava os 5 endereços entre as páginas, recebeu %d", len(seen))
	}
}

func TestListAddressesInvalidPaginationDefaults(t *testing.T) {
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	createAddressForSearch(t, "sao paulo", "SP", "01310-100", "avenida paulista", "1000")

	page := listAddresses(t, app, token, "page=0&limit=-1")
	if len(page.Data) != 1 || page.HasNext {
		t.Errorf("esperava os defaults 1/10, recebeu %d itens, has_next=%v", len(page.Data), page.HasNext)
	}
}

func TestListAddressesUnauthorized(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest("GET", "/v1/address", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
}
