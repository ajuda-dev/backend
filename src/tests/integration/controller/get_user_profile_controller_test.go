package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func doGetUserProfile(t *testing.T, app *fiber.App, userId string, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/user/"+userId, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func getUserProfileRawBody(t *testing.T, resp *http.Response) (string, map[string]any) {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return string(raw), body
}

func decodeUserProfileDtoOut(t *testing.T, resp *http.Response) dto.UserProfileDtoOut {
	t.Helper()
	defer resp.Body.Close()
	var respDto dto.UserProfileDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func updateUserProfile(t *testing.T, app *fiber.App, userId string, body []byte, token string) {
	t.Helper()
	resp := doPutUser(t, app, userId, body, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao preparar o perfil, recebeu %d", resp.StatusCode)
	}
}

func visibilityKeys(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	rawConfig, exists := body["configVisibility"]
	if !exists {
		t.Fatalf("esperava 'configVisibility' no body, recebeu %+v", body)
	}
	config, ok := rawConfig.(map[string]any)
	if !ok {
		t.Fatalf("esperava 'configVisibility' como objeto, recebeu %+v", rawConfig)
	}
	return config
}

func TestGetUserByIdSelfReturnsFullProfile(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "get_self@ajuda.dev", domain.UserRoleUser)
	token := validTokenFor(t, user.Id)
	updateUserProfile(t, app, user.Id, []byte(`{
		"description": "Desenvolvedor backend",
		"configVisibility": {
			"github":   { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"linkedin": { "value": "https://www.linkedin.com/in/devrocha/", "shareWithCommunity": false },
			"phone":    { "value": "+55 (11) 99999-9999", "shareWithCommunity": true }
		}
	}`), token)

	resp := doGetUserProfile(t, app, user.Id, token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no perfil do próprio usuário, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserProfileDtoOut(t, resp)
	if respDto.Id != user.Id {
		t.Errorf("esperava id '%s', recebeu '%s'", user.Id, respDto.Id)
	}
	if respDto.Name != user.Name {
		t.Errorf("esperava name '%s', recebeu '%s'", user.Name, respDto.Name)
	}
	if respDto.Description != "Desenvolvedor backend" {
		t.Errorf("esperava description 'Desenvolvedor backend', recebeu '%s'", respDto.Description)
	}
	if respDto.Email != user.Email {
		t.Errorf("esperava email '%s' para o próprio usuário, recebeu '%s'", user.Email, respDto.Email)
	}
	if !respDto.ConfigVisibility[domain.VisibilityKeyLinkedin].ShareWithCommunity {
		if _, exists := respDto.ConfigVisibility[domain.VisibilityKeyLinkedin]; !exists {
			t.Errorf("esperava linkedin presente no perfil completo, recebeu %+v", respDto.ConfigVisibility)
		}
	}
	if respDto.ConfigVisibility[domain.VisibilityKeyLinkedin].Value != "https://www.linkedin.com/in/devrocha/" {
		t.Errorf("esperava linkedin não compartilhado visível para o próprio, recebeu %+v", respDto.ConfigVisibility[domain.VisibilityKeyLinkedin])
	}
	emailConfig := respDto.ConfigVisibility[domain.VisibilityKeyEmail]
	if emailConfig.Value != user.Email {
		t.Errorf("esperava email.value '%s' espelhado, recebeu '%s'", user.Email, emailConfig.Value)
	}
}

func TestGetUserByIdAdminReturnsFullProfile(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_admin_target@ajuda.dev", domain.UserRoleUser)
	admin := createUserWithRole(t, "get_admin@ajuda.dev", domain.UserRoleAdmin)
	updateUserProfile(t, app, target.Id, []byte(`{
		"description": "Resumo do alvo",
		"configVisibility": {
			"github":   { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"linkedin": { "value": "https://www.linkedin.com/in/devrocha/", "shareWithCommunity": false }
		}
	}`), validTokenFor(t, target.Id))

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o admin lendo terceiro, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserProfileDtoOut(t, resp)
	if respDto.Email != target.Email {
		t.Errorf("esperava email '%s' para o admin, recebeu '%s'", target.Email, respDto.Email)
	}
	if respDto.Description != "Resumo do alvo" {
		t.Errorf("esperava description 'Resumo do alvo', recebeu '%s'", respDto.Description)
	}
	linkedin, exists := respDto.ConfigVisibility[domain.VisibilityKeyLinkedin]
	if !exists {
		t.Fatalf("esperava linkedin presente no perfil completo do admin, recebeu %+v", respDto.ConfigVisibility)
	}
	if linkedin.ShareWithCommunity {
		t.Errorf("esperava linkedin com shareWithCommunity false, recebeu %+v", linkedin)
	}
	if linkedin.Value != "https://www.linkedin.com/in/devrocha/" {
		t.Errorf("esperava linkedin com valor real, recebeu %+v", linkedin)
	}
}

func TestGetUserByIdThirdPartySeesOnlySharedFields(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_third_target@ajuda.dev", domain.UserRoleUser)
	other := createUserWithRole(t, "get_third_other@ajuda.dev", domain.UserRoleUser)
	updateUserProfile(t, app, target.Id, []byte(`{
		"description": "Resumo do alvo",
		"configVisibility": {
			"github":   { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"linkedin": { "value": "https://www.linkedin.com/in/devrocha/", "shareWithCommunity": false },
			"phone":    { "value": "+55 (11) 99999-9999", "shareWithCommunity": true }
		}
	}`), validTokenFor(t, target.Id))

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, other.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para terceiro lendo o perfil, recebeu %d", resp.StatusCode)
	}
	_, body := getUserProfileRawBody(t, resp)
	config := visibilityKeys(t, body)
	if _, exists := config[domain.VisibilityKeyGithub]; !exists {
		t.Errorf("esperava github compartilhado no mapa, recebeu %+v", config)
	}
	if _, exists := config[domain.VisibilityKeyPhone]; !exists {
		t.Errorf("esperava phone compartilhado no mapa, recebeu %+v", config)
	}
	if _, exists := config[domain.VisibilityKeyLinkedin]; exists {
		t.Errorf("esperava linkedin ausente do mapa para terceiro, recebeu %+v", config)
	}
	if body["name"] != target.Name {
		t.Errorf("esperava name '%s' para terceiro, recebeu '%v'", target.Name, body["name"])
	}
	if body["description"] != "Resumo do alvo" {
		t.Errorf("esperava description 'Resumo do alvo' para terceiro, recebeu '%v'", body["description"])
	}
}

func TestGetUserByIdThirdPartyWithoutSharedEmailDoesNotReceiveEmail(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_noemail_target@ajuda.dev", domain.UserRoleUser)
	other := createUserWithRole(t, "get_noemail_other@ajuda.dev", domain.UserRoleUser)
	updateUserProfile(t, app, target.Id, []byte(`{
		"configVisibility": {
			"github": { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"email":  { "shareWithCommunity": false }
		}
	}`), validTokenFor(t, target.Id))

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, other.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para terceiro lendo o perfil, recebeu %d", resp.StatusCode)
	}
	_, body := getUserProfileRawBody(t, resp)
	if _, exists := body["email"]; exists {
		t.Errorf("esperava email ausente no topo para terceiro, recebeu '%v'", body["email"])
	}
	config := visibilityKeys(t, body)
	if _, exists := config[domain.VisibilityKeyEmail]; exists {
		t.Errorf("esperava chave email ausente do mapa para terceiro, recebeu %+v", config)
	}
}

func TestGetUserByIdThirdPartyWithSharedEmailReceivesEmail(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_email_target@ajuda.dev", domain.UserRoleUser)
	other := createUserWithRole(t, "get_email_other@ajuda.dev", domain.UserRoleUser)
	updateUserProfile(t, app, target.Id, []byte(`{
		"configVisibility": {
			"github": { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"email":  { "shareWithCommunity": true }
		}
	}`), validTokenFor(t, target.Id))

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, other.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para terceiro lendo o perfil, recebeu %d", resp.StatusCode)
	}
	_, body := getUserProfileRawBody(t, resp)
	if body["email"] != target.Email {
		t.Errorf("esperava email '%s' no topo para terceiro, recebeu '%v'", target.Email, body["email"])
	}
	config := visibilityKeys(t, body)
	emailEntry, exists := config[domain.VisibilityKeyEmail].(map[string]any)
	if !exists {
		t.Fatalf("esperava chave email no mapa para terceiro, recebeu %+v", config)
	}
	if emailEntry["value"] != target.Email {
		t.Errorf("esperava email.value '%s' espelhado, recebeu '%v'", target.Email, emailEntry["value"])
	}
	if emailEntry["shareWithCommunity"] != true {
		t.Errorf("esperava email.shareWithCommunity true, recebeu '%v'", emailEntry["shareWithCommunity"])
	}
}

func TestGetUserByIdModeratorSeesOnlySharedFields(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_mod_target@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "get_mod@ajuda.dev", domain.UserRoleModerator)
	updateUserProfile(t, app, target.Id, []byte(`{
		"configVisibility": {
			"github":   { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"linkedin": { "value": "https://www.linkedin.com/in/devrocha/", "shareWithCommunity": false }
		}
	}`), validTokenFor(t, target.Id))

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para moderador lendo terceiro, recebeu %d", resp.StatusCode)
	}
	_, body := getUserProfileRawBody(t, resp)
	config := visibilityKeys(t, body)
	if _, exists := config[domain.VisibilityKeyGithub]; !exists {
		t.Errorf("esperava github compartilhado no mapa, recebeu %+v", config)
	}
	if _, exists := config[domain.VisibilityKeyLinkedin]; exists {
		t.Errorf("esperava linkedin ausente do mapa para moderador, recebeu %+v", config)
	}
	if _, exists := body["email"]; exists {
		t.Errorf("esperava email ausente no topo para moderador, recebeu '%v'", body["email"])
	}
}

func TestGetUserByIdNotFound(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	requester := createUserWithRole(t, "get_notfound@ajuda.dev", domain.UserRoleUser)

	resp := doGetUserProfile(t, app, uuidv7.New().String(), validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para usuário inexistente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "User not found" {
		t.Errorf("esperava message 'User not found', recebeu '%s'", respBody.Message)
	}
}

func TestGetUserByIdSoftDeletedNotFound(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_deleted_target@ajuda.dev", domain.UserRoleUser)
	requester := createUserWithRole(t, "get_deleted_requester@ajuda.dev", domain.UserRoleUser)
	if err := userRepository.SoftDeleteById(target.Id); err != nil {
		t.Fatalf("failed to soft delete user: %v", err)
	}

	resp := doGetUserProfile(t, app, target.Id, validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 para usuário arquivado, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "User not found" {
		t.Errorf("esperava message 'User not found', recebeu '%s'", respBody.Message)
	}
}

func TestGetUserByIdInvalidId(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	requester := createUserWithRole(t, "get_invalid@ajuda.dev", domain.UserRoleUser)
	token := validTokenFor(t, requester.Id)

	for _, invalidId := range []string{"abc", "123", "not-a-uuid"} {
		resp := doGetUserProfile(t, app, invalidId, token)
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("esperava 400 para id '%s', recebeu %d", invalidId, resp.StatusCode)
		}
		respBody := decodeRestErr(t, resp)
		causes := getCauseByField("userId", respBody.Causes)
		if len(causes) == 0 || causes[0] != "userId must be a valid UUID v7" {
			t.Errorf("esperava cause 'userId must be a valid UUID v7' para id '%s', recebeu %+v", invalidId, respBody.Causes)
		}
	}
}

func TestGetUserByIdWithoutToken(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithRole(t, "get_notoken_target@ajuda.dev", domain.UserRoleUser)

	resp := doGetUserProfile(t, app, target.Id, "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = doGetUserProfile(t, app, target.Id, validTokenFor(t, uuidv7.New().String()))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 com requester inexistente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func TestGetUserByIdNeverReturnsPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	target := createUserWithHashedPassword(t, app, "get_secret_target@ajuda.dev", domain.UserRoleUser, "123456")
	admin := createUserWithRole(t, "get_secret_admin@ajuda.dev", domain.UserRoleAdmin)
	other := createUserWithRole(t, "get_secret_other@ajuda.dev", domain.UserRoleUser)
	updateUserProfile(t, app, target.Id, []byte(`{
		"configVisibility": {
			"github": { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true }
		}
	}`), validTokenFor(t, target.Id))

	requesters := map[string]string{
		"próprio":  validTokenFor(t, target.Id),
		"admin":    validTokenFor(t, admin.Id),
		"terceiro": validTokenFor(t, other.Id),
	}
	for name, token := range requesters {
		resp := doGetUserProfile(t, app, target.Id, token)
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("esperava 200 para %s, recebeu %d", name, resp.StatusCode)
		}
		raw, _ := getUserProfileRawBody(t, resp)
		if strings.Contains(raw, "password") {
			t.Errorf("esperava body sem 'password' para %s, recebeu '%s'", name, raw)
		}
		if strings.Contains(raw, "$2a$") || strings.Contains(raw, "$2b$") {
			t.Errorf("esperava body sem hash de senha para %s, recebeu '%s'", name, raw)
		}
		if strings.Contains(raw, "\"role\"") {
			t.Errorf("esperava body sem 'role' para %s, recebeu '%s'", name, raw)
		}
		if strings.Contains(raw, "\"token\"") {
			t.Errorf("esperava body sem 'token' para %s, recebeu '%s'", name, raw)
		}
	}
}
