package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
	"github.com/gofiber/fiber/v2"
)






func TestCreateUserSuccess(t *testing.T) {
	db, cleanup, err := setupTestDB(context.Background())
	if err != nil {
		t.Fatalf("Erro ao configurar o banco de dados: %v", err)
	}
	defer cleanup()
	app := fiber.New()

	routes.SetupRoutesUser(app, controller.NewUserController(
		service.NewUserService(repository.NewUserRepository(db),
			validator.NewUserValidator())))
	body := []byte(`
	{
		"name": "teste",
		"email":"teste@ajuda.dev",
		"password": "123456"
	}
	`)
	req := httptest.NewRequest("POST", "/v1/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201, recebeu %d", resp.StatusCode)
	}

}

func TestCreateUserEmailAlreadyExists(t *testing.T) {
	db, cleanup, err := setupTestDB(context.Background())
	if err != nil {
		t.Fatalf("Erro ao configurar o banco de dados: %v", err)
	}
	defer cleanup()
	app := fiber.New()
	repository := repository.NewUserRepository(db)
	routes.SetupRoutesUser(app, controller.NewUserController(
		service.NewUserService(repository,
			validator.NewUserValidator())))
	body := []byte(`
	{
		"name": "teste",
		"email": "teste@ajuda.dev",
		"password": "123456"
	}
	`)
	// Primeiro cria o usuário
	_, createErr := repository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    "teste@ajuda.dev",
		Password:  "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
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
	if len(respBody.Causes) == 0 || respBody.Causes[0].Field != "email" || respBody.Causes[0].Message != "Email already exists" {
		t.Errorf("esperava cause para email já existente, recebeu %+v", respBody.Causes)
	}
}





