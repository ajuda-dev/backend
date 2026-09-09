package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samborkent/uuidv7"
)

type protectedRoute struct {
	method string
	path   string
}

func protectedRoutes() []protectedRoute {
	return []protectedRoute{
		{http.MethodPost, "/v1/address/register"},
		{http.MethodPost, "/v1/community/register"},
		{http.MethodGet, "/v1/community"},
		{http.MethodPost, "/v1/event/register"},
		{http.MethodGet, "/v1/event/invalid-id"},
		{http.MethodGet, "/v1/event"},
		{http.MethodDelete, "/v1/event/invalid-id"},
		{http.MethodPost, "/v1/event/invalid-id/join"},
		{http.MethodPost, "/v1/event/invalid-id/participants"},
		{http.MethodGet, "/v1/event/invalid-id/participants"},
		{http.MethodPut, "/v1/event/invalid-id/participants/invalid-id/status"},
		{http.MethodDelete, "/v1/event/invalid-id/participants/invalid-id"},
		{http.MethodPost, "/v1/skill/register"},
		{http.MethodGet, "/v1/skill/invalid-id"},
		{http.MethodGet, "/v1/skill"},
		{http.MethodPut, "/v1/skill/invalid-id"},
		{http.MethodDelete, "/v1/skill/invalid-id"},
		{http.MethodPost, "/v1/skill/invalid-id/users"},
		{http.MethodGet, "/v1/user"},
		{http.MethodGet, "/v1/user/invalid-id/skills"},
		{http.MethodDelete, "/v1/user/invalid-id/skills/invalid-id"},
	}
}

func tokenSignedWith(t *testing.T, claims jwt.RegisteredClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("erro ao assinar token: %v", err)
	}
	return signed
}

func expiredToken(t *testing.T) string {
	t.Helper()
	return tokenSignedWith(t, jwt.RegisteredClaims{
		Subject:   uuidv7.New().String(),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)),
	}, "test-secret")
}

func TestProtectedRoutesRejectMissingToken(t *testing.T) {
	app := setupApp()

	for _, route := range protectedRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(route.method, route.path, nil))
			if err != nil {
				t.Fatalf("erro ao executar requisição: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf("esperava 401, recebeu %d", resp.StatusCode)
			}
			var respBody rest_err.RestErr
			if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
				t.Fatalf("erro ao decodificar body: %v", err)
			}
			if respBody.Code != fiber.StatusUnauthorized || respBody.Err != "unauthorized" {
				t.Errorf("esperava err 'unauthorized' com code 401, recebeu %+v", respBody)
			}
		})
	}
}

func TestProtectedRoutesRejectInvalidToken(t *testing.T) {
	app := setupApp()

	tokens := []string{
		"abc",
		"Bearer abc",
		tokenSignedWith(t, jwt.RegisteredClaims{Subject: "user-1"}, "wrong-secret"),
	}

	for _, route := range protectedRoutes() {
		for _, token := range tokens {
			t.Run(route.method+" "+route.path+" com token inválido", func(t *testing.T) {
				req := httptest.NewRequest(route.method, route.path, nil)
				resp, err := doAuthedRequest(app, req, token)
				if err != nil {
					t.Fatalf("erro ao executar requisição: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != fiber.StatusUnauthorized {
					t.Errorf("esperava 401, recebeu %d", resp.StatusCode)
				}
				var respBody rest_err.RestErr
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("erro ao decodificar body: %v", err)
				}
				if respBody.Code != fiber.StatusUnauthorized || respBody.Err != "unauthorized" {
					t.Errorf("esperava err 'unauthorized' com code 401, recebeu %+v", respBody)
				}
			})
		}
	}
}

func TestSwaggerRouteIsPublic(t *testing.T) {
	app := setupApp()

	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json"} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatalf("erro ao executar requisição: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("esperava 200 em %s sem token, recebeu %d", path, resp.StatusCode)
		}
	}
}

func TestProtectedRouteRejectsExpiredToken(t *testing.T) {
	app := setupApp()

	req := httptest.NewRequest("GET", "/v1/community", nil)
	resp, err := doAuthedRequest(app, req, expiredToken(t))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "invalid or expired token" {
		t.Errorf("esperava message 'invalid or expired token', recebeu '%s'", respBody.Message)
	}
}
