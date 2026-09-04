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
	resp, err := app.Test(req)
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
	body := []byte(`
	{	}
	`)
	req := newCommunityRegisterRequest(body)
	resp, err := app.Test(req)
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
	if len(causes) == 0 || causes[0] != "OwnerId is not valid" {
		t.Errorf("esperava cause para o campo 'OwnerId', recebeu %+v", respBody.Causes)
	}
	
}
