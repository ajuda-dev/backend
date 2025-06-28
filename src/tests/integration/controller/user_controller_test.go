package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	db     *gorm.DB
	cleanupDB  func()
	userRepository  repository.UserRepository
	app *fiber.App
)

func TestMain(m *testing.M) {
	var err error
	db, cleanupDB, err = setupTestDB(context.Background())
	if err != nil {
			panic("Erro ao configurar o banco de dados: " + err.Error())
	}
	userRepository = repository.NewUserRepository(db)
	
	code := m.Run()
	cleanupDB()
	os.Exit(code)
}


func setupApp() *fiber.App {
	app := fiber.New()
	routes.SetupRoutesUser(app, controller.NewUserController(
			service.NewUserService(userRepository,
					validator.NewUserValidator())))
	return app
}

func newUserRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}


func TestCreateUserSuccess(t *testing.T) {
	app := setupApp()
	body := []byte(`
	{
		"name": "teste",
		"email":"teste@ajuda.dev",
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
	
	db.Exec("DELETE FROM users")
}


func TestCreateUserEmailAlreadyExists(t *testing.T) {
	
	app := fiber.New()
	repository := repository.NewUserRepository(db)
	routes.SetupRoutesUser(app, controller.NewUserController(
		service.NewUserService(repository,
			validator.NewUserValidator())))

	_, createErr := repository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    "teste@ajuda.dev",
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	body := []byte(`
	{
		"name": "teste",
		"email": "teste@ajuda.dev",
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
	if respBody.Err != "bad_request" {
		t.Errorf("esperava error 'bad_request', recebeu '%s'", respBody.Error())
	}
	if respBody.Code != 400 {
		t.Errorf("esperava code 400, recebeu %d", respBody.Code)
	}
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "email" || respBody.Causes[0].Message != "Email already exists" {
		t.Errorf("esperava cause para email já existente, recebeu %+v", respBody.Causes)
	}
	db.Exec("DELETE FROM users")
}


func TestCreateUserFail(t *testing.T) {
	app := fiber.New()

	routes.SetupRoutesUser(app, controller.NewUserController(
		service.NewUserService(repository.NewUserRepository(db),
			validator.NewUserValidator())))

	body := []byte(`
	{
		"name": "",
		"email": "",
		"password": ""
	}
	`)

	req := httptest.NewRequest("POST", "/v1/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
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
	if respBody.Err != "bad_request" {
		t.Errorf("esperava error 'bad_request', recebeu '%s'", respBody.Error())
	}
	if respBody.Code != 400 {
		t.Errorf("esperava code 400, recebeu %d", respBody.Code)
	}

	getCauseByField := func(field string, causesList []rest_err.Causes) []string {
		var messages []string
		for _, cause := range causesList {
			if cause.Field == field {
				messages = append(messages, cause.Message)
			}
		}
		return messages
	}

	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Name is not valid" {
		t.Errorf("esperava cause para o campo 'name', recebeu %+v", respBody.Causes)
	}
	causes = getCauseByField("email", respBody.Causes)
	expectedEmailMessages := map[string]bool{
		"Email cannot be empty": true,
		"Email is not valid":    true,
	}
	for _, msg := range causes {
		if !expectedEmailMessages[msg] {
			t.Errorf("mensagem inesperada para o campo 'email': %s", msg)
		}
	}
	causes = getCauseByField("password", respBody.Causes)
	expectedPassowrdMessages := map[string]bool{
		"Password must be at least 6 characters long": true,
		"Password cannot be empty":    true,
	}

	for _, msg := range causes {
		if !expectedPassowrdMessages[msg] {
			t.Errorf("mensagem inesperada para o campo 'password': %s", msg)
		}
	}
	db.Exec("DELETE FROM users")
}	