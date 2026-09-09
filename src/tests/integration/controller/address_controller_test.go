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
  		"zip_code": "test_zip_code"
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
			"street": "test_street"
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

}



func TestRegisterAddressAlreadyExist(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	app := setupApp()
	token := validTokenFor(t, uuidv7.New().String())
	addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
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


	resp, err = doAuthedRequest(app, req, token)
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


