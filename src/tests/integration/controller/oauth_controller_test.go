package controller_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	oauthTestFrontendURL = "http://frontend.test"
	oauthTestCallbackURL = "http://localhost:8080/v1/auth/github/callback"
	oauthTestTokenPrefix = oauthTestFrontendURL + "/auth/callback?token="
)

type fakeGithubServer struct {
	server         *httptest.Server
	providerUserId int64
	login          string
	name           string
	email          string
	emailPrimary   bool
	emailVerified  bool
}

func newFakeGithubServer(t *testing.T) *fakeGithubServer {
	t.Helper()
	fake := &fakeGithubServer{
		providerUserId: 42,
		login:          "octocat",
		name:           "Mona Lisa",
		email:          "mona@ajuda.dev",
		emailPrimary:   true,
		emailVerified:  true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.FormValue("code") != "valid-code" {
			io.WriteString(w, `{"error":"bad_verification_code","error_description":"The code passed is incorrect or expired."}`)
			return
		}
		io.WriteString(w, `{"access_token":"fake-access-token","token_type":"bearer"}`)
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`{"id":%d,"login":%q,"name":%q}`, fake.providerUserId, fake.login, fake.name))
	})
	mux.HandleFunc("/api/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`[{"email":%q,"primary":%t,"verified":%t}]`, fake.email, fake.emailPrimary, fake.emailVerified))
	})

	fake.server = httptest.NewServer(mux)
	t.Cleanup(fake.server.Close)
	return fake
}

func configureGithubProvider(t *testing.T, fake *fakeGithubServer) {
	t.Helper()
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GITHUB_CALLBACK_URL", oauthTestCallbackURL)
	t.Setenv("GITHUB_AUTHORIZE_URL", fake.server.URL+"/login/oauth/authorize")
	t.Setenv("GITHUB_TOKEN_URL", fake.server.URL+"/login/oauth/access_token")
	t.Setenv("GITHUB_API_BASE_URL", fake.server.URL+"/api")
	t.Setenv("OAUTH_FRONTEND_URL", oauthTestFrontendURL)
}

func startOAuthLogin(t *testing.T, app *fiber.App, provider string) (string, string, *http.Cookie) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/auth/"+provider+"/login", nil))
	if err != nil {
		t.Fatalf("erro ao iniciar o login oauth: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no login oauth, recebeu %d", response.StatusCode)
	}

	location := response.Header.Get("Location")
	parsedLocation, parseErr := url.Parse(location)
	if parseErr != nil {
		t.Fatalf("erro ao interpretar a URL de autorização: %v", parseErr)
	}
	state := parsedLocation.Query().Get("state")
	if state == "" {
		t.Fatalf("esperava state na URL de autorização, recebeu %s", location)
	}

	var stateCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "oauth_state_"+provider {
			stateCookie = cookie
			break
		}
	}
	if stateCookie == nil {
		t.Fatalf("esperava o cookie oauth_state_%s na resposta de login", provider)
	}
	return location, state, stateCookie
}

func completeOAuthCallback(t *testing.T, app *fiber.App, provider string, code string, state string, cookie *http.Cookie) *http.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet,
		"/v1/auth/"+provider+"/callback?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("erro ao executar o callback oauth: %v", err)
	}
	return response
}

func oauthTokenSubject(t *testing.T, app *fiber.App, provider string, code string) string {
	t.Helper()
	_, state, stateCookie := startOAuthLogin(t, app, provider)
	response := completeOAuthCallback(t, app, provider, code, state, stateCookie)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no callback oauth, recebeu %d", response.StatusCode)
	}
	location := response.Header.Get("Location")
	if !strings.HasPrefix(location, oauthTestTokenPrefix) {
		t.Fatalf("esperava redirect para o frontend com token, recebeu %s", location)
	}

	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(strings.TrimPrefix(location, oauthTestTokenPrefix), claims,
		func(*jwt.Token) (interface{}, error) { return []byte("test-secret"), nil })
	if err != nil || !parsedToken.Valid {
		t.Fatalf("esperava token JWT válido no redirect, erro: %v", err)
	}
	if claims.Subject == "" {
		t.Fatal("esperava subject no token JWT")
	}
	return claims.Subject
}

func expectOAuthBadRequest(t *testing.T, response *http.Response) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", response.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(response.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Code != fiber.StatusBadRequest || respBody.Err != "bad_request" {
		t.Fatalf("esperava err 'bad_request' com code 400, recebeu %+v", respBody)
	}
}

func countUsersByEmail(t *testing.T, email string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&entity.UserEntity{}).Where("email = ?", email).Count(&count).Error; err != nil {
		t.Fatalf("erro ao contar usuários: %v", err)
	}
	return count
}

func TestOAuthGithubLoginRedirectsToProvider(t *testing.T) {
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	location, _, stateCookie := startOAuthLogin(t, app, "github")
	if !strings.HasPrefix(location, fake.server.URL+"/login/oauth/authorize") {
		t.Errorf("esperava redirect para o authorize do provedor, recebeu %s", location)
	}
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("erro ao interpretar a URL de autorização: %v", err)
	}
	if clientId := parsedLocation.Query().Get("client_id"); clientId != "test-client-id" {
		t.Errorf("esperava client_id 'test-client-id', recebeu '%s'", clientId)
	}
	if redirectUri := parsedLocation.Query().Get("redirect_uri"); redirectUri != oauthTestCallbackURL {
		t.Errorf("esperava redirect_uri '%s', recebeu '%s'", oauthTestCallbackURL, redirectUri)
	}
	if scope := parsedLocation.Query().Get("scope"); scope != "user:email" {
		t.Errorf("esperava scope 'user:email', recebeu '%s'", scope)
	}
	if !stateCookie.HttpOnly {
		t.Error("esperava cookie de state com HttpOnly")
	}
}

func TestOAuthGithubCallbackCreatesUserAndReturnsToken(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	_, state, stateCookie := startOAuthLogin(t, app, "github")
	response := completeOAuthCallback(t, app, "github", "valid-code", state, stateCookie)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no callback oauth, recebeu %d", response.StatusCode)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "oauth_state_github" && cookie.Value != "" {
			t.Error("esperava o cookie de state limpo no callback")
		}
	}

	location := response.Header.Get("Location")
	if !strings.HasPrefix(location, oauthTestTokenPrefix) {
		t.Fatalf("esperava redirect para o frontend com token, recebeu %s", location)
	}
	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(strings.TrimPrefix(location, oauthTestTokenPrefix), claims,
		func(*jwt.Token) (interface{}, error) { return []byte("test-secret"), nil })
	if err != nil || !parsedToken.Valid {
		t.Fatalf("esperava token JWT válido no redirect, erro: %v", err)
	}

	var userEntity entity.UserEntity
	if err := db.Where("email = ?", fake.email).First(&userEntity).Error; err != nil {
		t.Fatalf("esperava usuário criado pelo oauth: %v", err)
	}
	if userEntity.Id != claims.Subject {
		t.Errorf("esperava subject '%s' no token, recebeu '%s'", userEntity.Id, claims.Subject)
	}
	if userEntity.Name != fake.name {
		t.Errorf("esperava name '%s', recebeu '%s'", fake.name, userEntity.Name)
	}
	if userEntity.Role != "USER" {
		t.Errorf("esperava role 'USER', recebeu '%s'", userEntity.Role)
	}
	if userEntity.Password != "" {
		t.Error("esperava usuário criado via oauth sem senha")
	}

	var accountEntity entity.OAuthAccountEntity
	if err := db.Where("provider = ? AND provider_user_id = ?", "github", "42").First(&accountEntity).Error; err != nil {
		t.Fatalf("esperava vínculo do usuário com o provedor: %v", err)
	}
	if accountEntity.UserId != userEntity.Id {
		t.Errorf("esperava vínculo com o usuário '%s', recebeu '%s'", userEntity.Id, accountEntity.UserId)
	}
	if accountEntity.Username != fake.login {
		t.Errorf("esperava username '%s' no vínculo, recebeu '%s'", fake.login, accountEntity.Username)
	}
	if accountEntity.Email != fake.email {
		t.Errorf("esperava email '%s' no vínculo, recebeu '%s'", fake.email, accountEntity.Email)
	}
}

func TestOAuthGithubLoginReusesLinkedAccount(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	firstSubject := oauthTokenSubject(t, app, "github", "valid-code")
	secondSubject := oauthTokenSubject(t, app, "github", "valid-code")
	if firstSubject != secondSubject {
		t.Errorf("esperava o mesmo usuário nos dois logins, recebeu '%s' e '%s'", firstSubject, secondSubject)
	}
	if usersCount := countUsersByEmail(t, fake.email); usersCount != 1 {
		t.Errorf("esperava 1 usuário com o email '%s', recebeu %d", fake.email, usersCount)
	}

	var accountsCount int64
	if err := db.Model(&entity.OAuthAccountEntity{}).Where("provider = ?", "github").Count(&accountsCount).Error; err != nil {
		t.Fatalf("erro ao contar vínculos: %v", err)
	}
	if accountsCount != 1 {
		t.Errorf("esperava 1 vínculo oauth, recebeu %d", accountsCount)
	}
}

func TestOAuthGithubLoginLinksExistingUserByEmail(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	fake.providerUserId = 99
	configureGithubProvider(t, fake)
	app := setupApp()

	existingUser := createUserWithRole(t, fake.email, "USER")
	subject := oauthTokenSubject(t, app, "github", "valid-code")
	if subject != existingUser.Id {
		t.Errorf("esperava o usuário existente '%s', recebeu '%s'", existingUser.Id, subject)
	}
	if usersCount := countUsersByEmail(t, fake.email); usersCount != 1 {
		t.Errorf("esperava nenhum usuário duplicado, recebeu %d usuários com o email '%s'", usersCount, fake.email)
	}

	var accountEntity entity.OAuthAccountEntity
	if err := db.Where("provider = ? AND provider_user_id = ?", "github", "99").First(&accountEntity).Error; err != nil {
		t.Fatalf("esperava vínculo criado para o usuário existente: %v", err)
	}
	if accountEntity.UserId != existingUser.Id {
		t.Errorf("esperava vínculo com o usuário '%s', recebeu '%s'", existingUser.Id, accountEntity.UserId)
	}
}

func TestOAuthCallbackRejectsInvalidState(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	_, state, stateCookie := startOAuthLogin(t, app, "github")

	expectOAuthBadRequest(t, completeOAuthCallback(t, app, "github", "valid-code", state+"tampered", stateCookie))
	expectOAuthBadRequest(t, completeOAuthCallback(t, app, "github", "valid-code", state, nil))

	if usersCount := countUsersByEmail(t, fake.email); usersCount != 0 {
		t.Errorf("esperava nenhum usuário criado com state inválido, recebeu %d", usersCount)
	}
}

func TestOAuthCallbackRedirectsToFrontendOnProviderFailure(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	for _, code := range []string{"invalid-code", ""} {
		_, state, stateCookie := startOAuthLogin(t, app, "github")
		response := completeOAuthCallback(t, app, "github", code, state, stateCookie)
		if response.StatusCode != fiber.StatusFound {
			response.Body.Close()
			t.Fatalf("esperava 302 no callback com code '%s', recebeu %d", code, response.StatusCode)
		}
		location := response.Header.Get("Location")
		response.Body.Close()
		if location != oauthTestFrontendURL+"?error=auth_failed" {
			t.Errorf("esperava redirect de erro para o frontend, recebeu %s", location)
		}
	}

	if usersCount := countUsersByEmail(t, fake.email); usersCount != 0 {
		t.Errorf("esperava nenhum usuário criado em falha do provedor, recebeu %d", usersCount)
	}
}

func TestOAuthCallbackRejectsUnverifiedEmail(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	fake.emailVerified = false
	configureGithubProvider(t, fake)
	app := setupApp()

	_, state, stateCookie := startOAuthLogin(t, app, "github")
	response := completeOAuthCallback(t, app, "github", "valid-code", state, stateCookie)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no callback, recebeu %d", response.StatusCode)
	}
	if location := response.Header.Get("Location"); location != oauthTestFrontendURL+"?error=auth_failed" {
		t.Errorf("esperava redirect de erro para o frontend, recebeu %s", location)
	}
	if usersCount := countUsersByEmail(t, fake.email); usersCount != 0 {
		t.Errorf("esperava nenhum usuário criado sem email verificado, recebeu %d", usersCount)
	}
}

func TestOAuthLoginRejectsUnknownProvider(t *testing.T) {
	app := setupApp()

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/auth/gitlab/login", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	expectOAuthBadRequest(t, response)
}

func TestOAuthLoginRejectsUnconfiguredProvider(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	app := setupApp()

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/auth/github/login", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	expectOAuthBadRequest(t, response)
}
