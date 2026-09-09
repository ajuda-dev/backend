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
