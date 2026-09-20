package routes

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSetupHealthRoute(t *testing.T) {
	app := fiber.New()
	SetupHealthRoute(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSetupSwaggerRouteDisabled(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "false")
	app := fiber.New()
	SetupSwaggerRoute(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/swagger/index.html", nil))
	if err != nil {
		t.Fatalf("swagger request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 when swagger is disabled, got %d", resp.StatusCode)
	}
}

func TestSwaggerEnabledDefault(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "")
	if !swaggerEnabled() {
		t.Fatal("empty SWAGGER_ENABLED should keep swagger on")
	}
}

func TestSwaggerEnabledFalse(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "false")
	if swaggerEnabled() {
		t.Fatal("SWAGGER_ENABLED=false should disable swagger")
	}
}
