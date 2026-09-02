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








func newUserRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func getCauseByField(field string, causesList []rest_err.Causes) []string {
	var messages []string
	for _, cause := range causesList {
		if cause.Field == field {
			messages = append(messages, cause.Message)
		}
	}
	return messages
}

func TestCreateUserSuccess(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	
	body := []byte(`
	{
		"name": "teste",
		"email":"` + testEmail + `",
		"password": "123456"
	}
	`)
	req := newUserRegisterRequest(body)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}
	var respBody struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Token == "" {
		t.Error("esperava token não vazio na resposta")
	}
	user, findUserError := userRepository.GetUserByEmail(testEmail)
	if findUserError != nil || user == nil   || user.Id == "" {
		t.Fatalf("user not found in database: %v", findUserError)
	}
}

func TestCreateUserEmailAlreadyExists(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()

	_, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	body := []byte(`
	{
		"name": "teste",
		"email": "` + testEmail + `",
		"password": "123456"
	}
	`)
	req := newUserRegisterRequest(body)
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
	if respBody.Message != "Invalid user data" {
		t.Errorf("esperava message 'Invalid user data', recebeu '%s'", respBody.Message)
	}
	verifyCodeError(t, respBody)
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "email" || respBody.Causes[0].Message != "Email already exists" {
		t.Errorf("esperava cause para email já existente, recebeu %+v", respBody.Causes)
	}
}

func TestCreateUserFail(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()

	body := []byte(`
	{
		"name": "",
		"email": "",
		"password": ""
	}
	`)

	req := newUserRegisterRequest(body)
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
	if respBody.Message != "Invalid user data" {
		t.Errorf("esperava message 'Invalid user data', recebeu '%s'", respBody.Message)
	}
	verifyCodeError(t, respBody)

	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Name is not valid" {
		t.Errorf("esperava cause para o campo 'name', recebeu %+v", respBody.Causes)
	}
	verifyEmailFieldError(t, respBody)
	verifyPasswordFieldError(t, respBody)

	
}




func verifyEmailFieldError(t *testing.T, respBody rest_err.RestErr) {
	causes := getCauseByField("email", respBody.Causes)
	expectedEmailMessages := map[string]bool{
		"Email cannot be empty": true,
		"Email is not valid":    true,
	}

	for expectedMsg := range expectedEmailMessages {
		found := false
		for _, msg := range causes {
			if msg == expectedMsg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("esperava mensagem '%s' para o campo 'email', mas não foi encontrada. Mensagens recebidas: %+v", expectedMsg, causes)
		}
	}
}


func verifyPasswordFieldError(t *testing.T, respBody rest_err.RestErr) {
	causes := getCauseByField("password", respBody.Causes)
	expectedPasswordMessages := map[string]bool{
		"Password must be at least 6 characters long": true,
		"Password cannot be empty":                    true,
	}
	// Verifica se todas as mensagens esperadas estão presentes
	for expectedMsg := range expectedPasswordMessages {
		found := false
		for _, msg := range causes {
			if msg == expectedMsg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("esperava mensagem '%s' para o campo 'password', mas não foi encontrada. Mensagens recebidas: %+v", expectedMsg, causes)
		}
	}
}
