package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
)

func newRequestWithSessionCookie(method string, path string, token string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	return req
}

func TestProtectedRouteAcceptsSessionCookie(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "session_cookie@ajuda.dev", domain.UserRoleUser)

	resp, err := app.Test(newRequestWithSessionCookie(http.MethodGet, "/v1/community", validTokenFor(t, user.Id)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 com cookie de sessão válido, recebeu %d", resp.StatusCode)
	}
}

func TestProtectedRouteRejectsInvalidSessionCookieAndClearsIt(t *testing.T) {
	app := setupApp()

	resp, err := app.Test(newRequestWithSessionCookie(http.MethodGet, "/v1/community", "not-a-jwt"))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 com cookie inválido, recebeu %d", resp.StatusCode)
	}
	cookie := sessionCookieFrom(t, resp)
	if cookie == nil || cookie.Value != "" {
		t.Error("esperava o cookie de sessão limpo na resposta 401")
	}
}

func TestMeRequiresAuthentication(t *testing.T) {
	app := setupApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/user/me", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no /v1/user/me sem sessão, recebeu %d", resp.StatusCode)
	}
}

func TestMeReturnsAuthenticatedUserFromSessionCookie(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "session_me_cookie@ajuda.dev", domain.UserRoleUser)

	resp, err := app.Test(newRequestWithSessionCookie(http.MethodGet, "/v1/user/me", validTokenFor(t, user.Id)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no /v1/user/me com cookie, recebeu %d", resp.StatusCode)
	}
	var respDto dto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.Id != user.Id || respDto.Name != user.Name || respDto.Email != user.Email || respDto.Role != domain.UserRoleUser {
		t.Errorf("esperava o usuário da sessão no /v1/user/me, recebeu %+v", respDto)
	}
	if !respDto.EmailVerified {
		t.Error("esperava emailVerified true no /v1/user/me")
	}
	if respDto.Token != "" {
		t.Errorf("esperava response sem token, recebeu '%s'", respDto.Token)
	}
}

func TestMeReturnsAuthenticatedUserFromBearerToken(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "session_me_bearer@ajuda.dev", domain.UserRoleAdmin)

	resp, err := doAuthedRequest(app, httptest.NewRequest(http.MethodGet, "/v1/user/me", nil), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no /v1/user/me com Bearer, recebeu %d", resp.StatusCode)
	}
	var respDto dto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.Id != user.Id || respDto.Role != domain.UserRoleAdmin {
		t.Errorf("esperava o usuário da sessão no /v1/user/me, recebeu %+v", respDto)
	}
}

func TestLogoutClearsSessionCookie(t *testing.T) {
	app := setupApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/v1/user/logout", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no logout, recebeu %d", resp.StatusCode)
	}
	cookie := sessionCookieFrom(t, resp)
	if cookie == nil || cookie.Value != "" {
		t.Errorf("esperava o cookie de sessão limpo no logout, recebeu %+v", cookie)
	}
}

func TestNativeClientLoginReturnsTokenInBody(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "session_native@ajuda.dev", domain.UserRoleUser, "123456")

	req := newUserLoginRequest([]byte(`{"email": "` + user.Email + `", "password": "123456"}`))
	req.Header.Set(middleware.NativeClientHeader, "native")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no login nativo, recebeu %d", resp.StatusCode)
	}
	var respDto dto.LoginUserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.Token == "" {
		t.Fatal("esperava token no corpo para cliente nativo")
	}
	if respDto.Id != user.Id || respDto.Role != domain.UserRoleUser {
		t.Errorf("esperava id/role do usuário no login nativo, recebeu %+v", respDto)
	}
	if cookie := sessionCookieFrom(t, resp); cookie == nil {
		t.Error("esperava cookie de sessão gravado também para cliente nativo")
	}

	meResp, err := doAuthedRequest(app, httptest.NewRequest(http.MethodGet, "/v1/user/me", nil), respDto.Token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava o token do corpo autenticando via Bearer, recebeu %d", meResp.StatusCode)
	}
}
