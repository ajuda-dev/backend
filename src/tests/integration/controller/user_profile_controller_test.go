package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"

	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

func rawConfigVisibility(t *testing.T, userId string) string {
	t.Helper()
	var raw *string
	if err := db.Raw("SELECT config_visibility::text FROM users WHERE id = ?", userId).Scan(&raw).Error; err != nil {
		t.Fatalf("failed to read config_visibility: %v", err)
	}
	if raw == nil {
		return ""
	}
	return *raw
}

func rawDescription(t *testing.T, userId string) string {
	t.Helper()
	var description string
	if err := db.Raw("SELECT description FROM users WHERE id = ?", userId).Scan(&description).Error; err != nil {
		t.Fatalf("failed to read description: %v", err)
	}
	return description
}

func decodeRawUserDtoOut(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return body
}

func TestUpdateUserProfileSuccess(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_success@ajuda.dev", userdomain.UserRoleUser)

	body := []byte(`{
		"name": "Lucas Rocha",
		"description": "Desenvolvedor backend",
		"configVisibility": {
			"github":    { "value": "https://github.com/LucasFreitasRocha", "shareWithCommunity": true },
			"linkedin":  { "value": "https://www.linkedin.com/in/devrocha/", "shareWithCommunity": true },
			"otherlink": { "value": "https://linktr.ee/devrocha", "shareWithCommunity": true },
			"photo":     { "value": "https://avatars.githubusercontent.com/u/33586465?s=400&v=4", "shareWithCommunity": true },
			"phone":     { "value": "+55 (11) 99999-9999", "shareWithCommunity": false },
			"email":     { "shareWithCommunity": false }
		}
	}`)
	resp := doPutUser(t, app, user.Id, body, validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no update do perfil, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Description != "Desenvolvedor backend" {
		t.Errorf("esperava description 'Desenvolvedor backend' na resposta, recebeu '%s'", respDto.Description)
	}
	if respDto.ConfigVisibility[userdomain.VisibilityKeyGithub].Value != "https://github.com/LucasFreitasRocha" {
		t.Errorf("esperava github na resposta, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyGithub])
	}
	if !respDto.ConfigVisibility[userdomain.VisibilityKeyGithub].ShareWithCommunity {
		t.Errorf("esperava github com shareWithCommunity true na resposta, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyGithub])
	}
	if respDto.ConfigVisibility[userdomain.VisibilityKeyPhone].Value != "+55 (11) 99999-9999" {
		t.Errorf("esperava phone na resposta, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyPhone])
	}
	emailConfig := respDto.ConfigVisibility[userdomain.VisibilityKeyEmail]
	if emailConfig.Value != user.Email {
		t.Errorf("esperava email.value '%s' espelhado na resposta, recebeu '%s'", user.Email, emailConfig.Value)
	}
	if emailConfig.ShareWithCommunity {
		t.Errorf("esperava email.shareWithCommunity false na resposta, recebeu %+v", emailConfig)
	}

	raw := rawConfigVisibility(t, user.Id)
	if raw == "" {
		t.Fatalf("esperava config_visibility persistida no banco, recebeu NULL")
	}
	var persisted map[string]userdomain.VisibilityConfig
	if err := json.Unmarshal([]byte(raw), &persisted); err != nil {
		t.Fatalf("erro ao decodificar config_visibility do banco: %v", err)
	}
	if persisted[userdomain.VisibilityKeyLinkedin].Value != "https://www.linkedin.com/in/devrocha/" {
		t.Errorf("esperava linkedin persistido no jsonb, recebeu %+v", persisted[userdomain.VisibilityKeyLinkedin])
	}
	if persisted[userdomain.VisibilityKeyEmail].Value != user.Email {
		t.Errorf("esperava email espelhado no jsonb, recebeu %+v", persisted[userdomain.VisibilityKeyEmail])
	}
	if rawDescription(t, user.Id) != "Desenvolvedor backend" {
		t.Errorf("esperava description persistida, recebeu '%s'", rawDescription(t, user.Id))
	}
}

func TestUpdateUserProfilePartialMerge(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_merge@ajuda.dev", userdomain.UserRoleUser)
	token := validTokenFor(t, user.Id)

	first := []byte(`{
		"configVisibility": {
			"github":   { "value": "https://github.com/antigo", "shareWithCommunity": true },
			"linkedin": { "value": "https://www.linkedin.com/in/antigo/", "shareWithCommunity": true }
		}
	}`)
	resp := doPutUser(t, app, user.Id, first, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no primeiro update, recebeu %d", resp.StatusCode)
	}

	second := []byte(`{
		"configVisibility": {
			"github": { "value": "https://github.com/novo", "shareWithCommunity": false }
		}
	}`)
	resp = doPutUser(t, app, user.Id, second, token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no update parcial, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.ConfigVisibility[userdomain.VisibilityKeyGithub].Value != "https://github.com/novo" {
		t.Errorf("esperava github atualizado, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyGithub])
	}
	if respDto.ConfigVisibility[userdomain.VisibilityKeyGithub].ShareWithCommunity {
		t.Errorf("esperava github com shareWithCommunity false, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyGithub])
	}
	if respDto.ConfigVisibility[userdomain.VisibilityKeyLinkedin].Value != "https://www.linkedin.com/in/antigo/" {
		t.Errorf("esperava linkedin intacto, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyLinkedin])
	}

	var persisted map[string]userdomain.VisibilityConfig
	if err := json.Unmarshal([]byte(rawConfigVisibility(t, user.Id)), &persisted); err != nil {
		t.Fatalf("erro ao decodificar config_visibility do banco: %v", err)
	}
	if persisted[userdomain.VisibilityKeyLinkedin].Value != "https://www.linkedin.com/in/antigo/" {
		t.Errorf("esperava linkedin intacto no jsonb, recebeu %+v", persisted[userdomain.VisibilityKeyLinkedin])
	}
	if persisted[userdomain.VisibilityKeyEmail].Value != user.Email {
		t.Errorf("esperava email espelhado no jsonb, recebeu %+v", persisted[userdomain.VisibilityKeyEmail])
	}
}

func TestUpdateUserProfileClearLink(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_clear@ajuda.dev", userdomain.UserRoleUser)
	token := validTokenFor(t, user.Id)

	first := []byte(`{
		"configVisibility": {
			"github": { "value": "https://github.com/antigo", "shareWithCommunity": true }
		}
	}`)
	resp := doPutUser(t, app, user.Id, first, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no primeiro update, recebeu %d", resp.StatusCode)
	}

	second := []byte(`{
		"configVisibility": {
			"github": { "value": "", "shareWithCommunity": false }
		}
	}`)
	resp = doPutUser(t, app, user.Id, second, token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao limpar o github, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.ConfigVisibility[userdomain.VisibilityKeyGithub].Value != "" {
		t.Errorf("esperava github limpo na resposta, recebeu %+v", respDto.ConfigVisibility[userdomain.VisibilityKeyGithub])
	}

	var persisted map[string]userdomain.VisibilityConfig
	if err := json.Unmarshal([]byte(rawConfigVisibility(t, user.Id)), &persisted); err != nil {
		t.Fatalf("erro ao decodificar config_visibility do banco: %v", err)
	}
	if persisted[userdomain.VisibilityKeyGithub].Value != "" {
		t.Errorf("esperava github limpo no jsonb, recebeu %+v", persisted[userdomain.VisibilityKeyGithub])
	}
}

func TestUpdateUserDescription(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_desc@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"description": "  Resumo do usuário  "}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no update só com description, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Description != "Resumo do usuário" {
		t.Errorf("esperava description 'Resumo do usuário' na resposta, recebeu '%s'", respDto.Description)
	}
	if respDto.Name != user.Name {
		t.Errorf("esperava name '%s' preservado, recebeu '%s'", user.Name, respDto.Name)
	}
	if rawDescription(t, user.Id) != "Resumo do usuário" {
		t.Errorf("esperava description persistida, recebeu '%s'", rawDescription(t, user.Id))
	}
}

func TestUpdateUserRejectsInvalidURL(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_badurl@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"github": {"value": "github.com/lucas", "shareWithCommunity": true}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 na url inválida, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("config_visibility.github.value", respBody.Causes)
	if len(causes) == 0 || causes[0] != "value must be a valid http or https url" {
		t.Errorf("esperava cause 'value must be a valid http or https url', recebeu %+v", respBody.Causes)
	}
	if rawConfigVisibility(t, user.Id) != "" {
		t.Errorf("esperava config_visibility NULL após 400, recebeu '%s'", rawConfigVisibility(t, user.Id))
	}
}

func TestUpdateUserRejectsInvalidPhone(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_badphone@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"phone": {"value": "123", "shareWithCommunity": false}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no telefone inválido, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("config_visibility.phone.value", respBody.Causes)
	if len(causes) == 0 || causes[0] != "value must be a valid phone number" {
		t.Errorf("esperava cause 'value must be a valid phone number', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserRejectsShareWithoutValue(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_share@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"linkedin": {"value": "", "shareWithCommunity": true}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no share sem valor, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("config_visibility.linkedin", respBody.Causes)
	if len(causes) == 0 || causes[0] != "shareWithCommunity requires a value" {
		t.Errorf("esperava cause 'shareWithCommunity requires a value', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserRejectsUnknownVisibilityKey(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_unknown@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"twitter": {"value": "https://x.com/lucas", "shareWithCommunity": true}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 na chave desconhecida, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("config_visibility.twitter", respBody.Causes)
	if len(causes) == 0 || causes[0] != "unsupported visibility key" {
		t.Errorf("esperava cause 'unsupported visibility key', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserRejectsEmailValue(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_emailvalue@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"email": {"value": "outro@ajudadev.dev", "shareWithCommunity": true}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no email.value enviado pelo cliente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("config_visibility.email", respBody.Causes)
	if len(causes) == 0 || causes[0] != "email value is managed by the system" {
		t.Errorf("esperava cause 'email value is managed by the system', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserAcceptsEmailShareTrue(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "profile_emailshare@ajuda.dev", userdomain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"configVisibility": {"email": {"shareWithCommunity": true}}}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no email com shareWithCommunity true, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	emailConfig := respDto.ConfigVisibility[userdomain.VisibilityKeyEmail]
	if !emailConfig.ShareWithCommunity {
		t.Errorf("esperava email.shareWithCommunity true na resposta, recebeu %+v", emailConfig)
	}
	if emailConfig.Value != user.Email {
		t.Errorf("esperava email.value '%s' espelhado, recebeu '%s'", user.Email, emailConfig.Value)
	}

	var persisted map[string]userdomain.VisibilityConfig
	if err := json.Unmarshal([]byte(rawConfigVisibility(t, user.Id)), &persisted); err != nil {
		t.Fatalf("erro ao decodificar config_visibility do banco: %v", err)
	}
	if !persisted[userdomain.VisibilityKeyEmail].ShareWithCommunity {
		t.Errorf("esperava email.shareWithCommunity true persistido, recebeu %+v", persisted[userdomain.VisibilityKeyEmail])
	}
	if persisted[userdomain.VisibilityKeyEmail].Value != user.Email {
		t.Errorf("esperava email.value '%s' persistido, recebeu '%s'", user.Email, persisted[userdomain.VisibilityKeyEmail].Value)
	}
}

func TestRegisterUserIgnoresProfileFields(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	body := []byte(`{
		"name": "teste perfil",
		"email": "profile_register@ajuda.dev",
		"password": "123456",
		"description": "não deveria ser aceito",
		"configVisibility": {
			"github": { "value": "https://github.com/lucas", "shareWithCommunity": true }
		}
	}`)
	resp, err := app.Test(newUserRegisterRequest(body))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no register, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRawUserDtoOut(t, resp)
	if _, exists := respBody["description"]; exists {
		t.Errorf("esperava response do register sem 'description', recebeu %+v", respBody)
	}
	if _, exists := respBody["configVisibility"]; exists {
		t.Errorf("esperava response do register sem 'configVisibility', recebeu %+v", respBody)
	}

	user, findErr := userRepository.GetUserByEmail("profile_register@ajuda.dev")
	if findErr != nil {
		t.Fatalf("failed to find registered user: %v", findErr)
	}
	if rawConfigVisibility(t, user.Id) != "" {
		t.Errorf("esperava config_visibility NULL no banco após o register, recebeu '%s'", rawConfigVisibility(t, user.Id))
	}
	if rawDescription(t, user.Id) != "" {
		t.Errorf("esperava description vazia no banco após o register, recebeu '%s'", rawDescription(t, user.Id))
	}
}

func TestLoginUserDoesNotReturnProfile(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithHashedPassword(t, app, "profile_login@ajuda.dev", userdomain.UserRoleUser, "123456")

	updateBody := []byte(`{
		"description": "Resumo",
		"configVisibility": {
			"github": { "value": "https://github.com/lucas", "shareWithCommunity": true }
		}
	}`)
	resp := doPutUser(t, app, user.Id, updateBody, validTokenFor(t, user.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no update do perfil, recebeu %d", resp.StatusCode)
	}

	loginBody := []byte(`{
		"email": "profile_login@ajuda.dev",
		"password": "123456"
	}`)
	respLogin, err := app.Test(newUserLoginRequest(loginBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if respLogin.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no login, recebeu %d", respLogin.StatusCode)
	}
	loginBodyOut := decodeRawUserDtoOut(t, respLogin)
	if _, exists := loginBodyOut["description"]; exists {
		t.Errorf("esperava response do login sem 'description', recebeu %+v", loginBodyOut)
	}
	if _, exists := loginBodyOut["configVisibility"]; exists {
		t.Errorf("esperava response do login sem 'configVisibility', recebeu %+v", loginBodyOut)
	}
	var loginDto userdto.LoginUserDtoOut
	raw, err := json.Marshal(loginBodyOut)
	if err != nil {
		t.Fatalf("erro ao remarshalar body do login: %v", err)
	}
	if err := json.Unmarshal(raw, &loginDto); err != nil {
		t.Fatalf("erro ao decodificar body do login: %v", err)
	}
	if loginDto.Token != "" || loginDto.Id != user.Id || loginDto.Email != user.Email || loginDto.Role != userdomain.UserRoleUser {
		t.Errorf("esperava id/email/role e token ausente no login (navegador usa cookie), recebeu %+v", loginDto)
	}
	if !loginDto.EmailVerified {
		t.Error("esperava emailVerified true no login")
	}
}
