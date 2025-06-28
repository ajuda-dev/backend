package controller_test

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/repository"
	"github.com/ajuda-dev/backend/src/service"
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






