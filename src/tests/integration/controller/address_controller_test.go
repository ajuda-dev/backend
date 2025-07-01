package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
)

func TestRegisterAddressSuccess(t *testing.T) {
	app := setupApp()
	body := []byte(
		`
	 	{
  		"zip_code": "test_zip_code"
		}
	 `)
	req := newAddressRegisterRequest(body)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}

	cleanUsersTable()

}

func newAddressRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/address/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}


func TestRegisterAddressBadRequest(t *testing.T) {
	app := setupApp()
	body := []byte(
		`
	 	{
			"city": "test_city",
			"state": "test_state",
			"street": "test_street"
		}
	 `)
	req := newAddressRegisterRequest(body)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400, recebeu %d", resp.StatusCode)
	}

	cleanUsersTable()
}



func TestRegisterAddressAlreadyExist(t *testing.T) {
	app := setupApp()
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

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()


	resp, err = app.Test(req)
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

	cleanUsersTable()
}


