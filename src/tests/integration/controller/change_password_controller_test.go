package controller_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

func newChangePasswordRequest(body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/user/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestChangePasswordRequiresAuthentication(t *testing.T) {
	app := setupApp()

	resp, err := app.Test(newChangePasswordRequest([]byte(`{"currentPassword":"antiga123","newPassword":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem sessão, recebeu %d", resp.StatusCode)
	}
}

func TestChangePasswordSuccessWithBearerAllowsLoginWithNewPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "change_bearer@ajuda.dev", userdomain.UserRoleUser, "antiga123")

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"currentPassword":"antiga123","newPassword":"nova456"}`)), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 204, recebeu %d body=%s", resp.StatusCode, body)
	}

	oldLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"change_bearer@ajuda.dev","password":"antiga123"}`)))
	if err != nil {
		t.Fatalf("erro no login antigo: %v", err)
	}
	defer oldLogin.Body.Close()
	if oldLogin.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("senha antiga deveria falhar, recebeu %d", oldLogin.StatusCode)
	}

	newLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"change_bearer@ajuda.dev","password":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro no login novo: %v", err)
	}
	defer newLogin.Body.Close()
	if newLogin.StatusCode != fiber.StatusOK {
		t.Errorf("esperava login com nova senha 200, recebeu %d", newLogin.StatusCode)
	}
}

func TestChangePasswordSuccessWithSessionCookie(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "change_cookie@ajuda.dev", userdomain.UserRoleUser, "antiga123")

	req := newChangePasswordRequest([]byte(`{"currentPassword":"antiga123","newPassword":"nova456"}`))
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: validTokenFor(t, user.Id)})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 204 com cookie de sessão, recebeu %d body=%s", resp.StatusCode, body)
	}

	newLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"change_cookie@ajuda.dev","password":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro no login novo: %v", err)
	}
	defer newLogin.Body.Close()
	if newLogin.StatusCode != fiber.StatusOK {
		t.Errorf("esperava login com nova senha 200, recebeu %d", newLogin.StatusCode)
	}
}

func TestChangePasswordWrongCurrentPasswordReturns401(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "change_wrong@ajuda.dev", userdomain.UserRoleUser, "antiga123")

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"currentPassword":"nao-e-essa","newPassword":"nova456"}`)), validTokenFor(t, user.Id))
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
	if respBody.Message != "invalid credentials" {
		t.Errorf("esperava message 'invalid credentials', recebeu '%s'", respBody.Message)
	}

	oldLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"change_wrong@ajuda.dev","password":"antiga123"}`)))
	if err != nil {
		t.Fatalf("erro no login antigo: %v", err)
	}
	defer oldLogin.Body.Close()
	if oldLogin.StatusCode != fiber.StatusOK {
		t.Errorf("senha original deveria continuar válida, recebeu %d", oldLogin.StatusCode)
	}
}

func TestChangePasswordShortNewPasswordReturns400(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "change_short@ajuda.dev", userdomain.UserRoleUser, "antiga123")

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"currentPassword":"antiga123","newPassword":"123"}`)), validTokenFor(t, user.Id))
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
	causes := getCauseByField("newPassword", respBody.Causes)
	if len(causes) != 1 || causes[0] != "Password must be at least 6 characters long" {
		t.Errorf("esperava cause de newPassword curta, recebeu %+v", respBody.Causes)
	}
}

func TestChangePasswordOAuthUserSetsFirstPasswordWithoutCurrentPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "oauth only",
		Email:    "change_oauth@ajuda.dev",
		Password: "",
	})
	if createErr != nil {
		t.Fatalf("failed to create oauth user: %v", createErr)
	}

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"newPassword":"nova456"}`)), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 204 para conta GitHub/OAuth definir senha, recebeu %d body=%s", resp.StatusCode, body)
	}

	login, err := app.Test(newUserLoginRequest([]byte(`{"email":"change_oauth@ajuda.dev","password":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro no login com senha recém-definida: %v", err)
	}
	defer login.Body.Close()
	if login.StatusCode != fiber.StatusOK {
		t.Errorf("esperava login com a senha definida 200, recebeu %d", login.StatusCode)
	}

	second, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"newPassword":"outra789"}`)), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro na segunda troca sem currentPassword: %v", err)
	}
	defer second.Body.Close()
	if second.StatusCode != fiber.StatusBadRequest {
		t.Errorf("depois de ter senha, currentPassword deveria ser obrigatório, recebeu %d", second.StatusCode)
	}
}

func TestChangePasswordOAuthUserIgnoresCurrentPasswordWhenSettingFirst(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "oauth dummy current",
		Email:    "change_oauth_dummy@ajuda.dev",
		Password: "",
	})
	if createErr != nil {
		t.Fatalf("failed to create oauth user: %v", createErr)
	}

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"currentPassword":"qualquer","newPassword":"nova456"}`)), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 204 mesmo com currentPassword dummy, recebeu %d body=%s", resp.StatusCode, body)
	}
}

func TestChangePasswordOAuthUserStillValidatesNewPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "oauth short",
		Email:    "change_oauth_short@ajuda.dev",
		Password: "",
	})
	if createErr != nil {
		t.Fatalf("failed to create oauth user: %v", createErr)
	}

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"newPassword":"123"}`)), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400, recebeu %d", resp.StatusCode)
	}
}

func TestChangePasswordOnlyAffectsAuthenticatedUser(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	owner := createUserWithHashedPassword(t, app, "change_owner@ajuda.dev", userdomain.UserRoleUser, "owner123")
	other := createUserWithHashedPassword(t, app, "change_other@ajuda.dev", userdomain.UserRoleUser, "other123")

	resp, err := doAuthedRequest(app, newChangePasswordRequest(
		[]byte(`{"currentPassword":"other123","newPassword":"nova456"}`)), validTokenFor(t, owner.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("senha de outro usuário não deveria passar, recebeu %d", resp.StatusCode)
	}

	otherLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"` + other.Email + `","password":"other123"}`)))
	if err != nil {
		t.Fatalf("erro no login do outro usuário: %v", err)
	}
	defer otherLogin.Body.Close()
	if otherLogin.StatusCode != fiber.StatusOK {
		t.Errorf("senha do outro usuário não deveria mudar, recebeu %d", otherLogin.StatusCode)
	}

	ownerLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"` + owner.Email + `","password":"owner123"}`)))
	if err != nil {
		t.Fatalf("erro no login do dono: %v", err)
	}
	defer ownerLogin.Body.Close()
	if ownerLogin.StatusCode != fiber.StatusOK {
		t.Errorf("senha do autenticado não deveria mudar com currentPassword alheia, recebeu %d", ownerLogin.StatusCode)
	}
}
