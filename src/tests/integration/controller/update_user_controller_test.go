package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
	"golang.org/x/crypto/bcrypt"
)

func newUpdateUserRequest(userId string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/v1/user/"+userId, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func doPutUser(t *testing.T, app *fiber.App, userId string, body []byte, token string) *http.Response {
	t.Helper()
	req := newUpdateUserRequest(userId, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func decodeUserDtoOut(t *testing.T, resp *http.Response) dto.UserDtoOut {
	t.Helper()
	defer resp.Body.Close()
	var respDto dto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func userUpdatedAt(t *testing.T, userId string) time.Time {
	t.Helper()
	var updatedAt time.Time
	if err := db.Raw("SELECT updated_at FROM users WHERE id = ?", userId).Scan(&updatedAt).Error; err != nil {
		t.Fatalf("failed to read updated_at: %v", err)
	}
	return updatedAt
}

func createUserWithHashedPassword(t *testing.T, app *fiber.App, email string, role string, password string) *domain.UserDomain {
	t.Helper()
	user := createUserWithRole(t, email, role)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if dbErr := db.Exec("UPDATE users SET password = ? WHERE id = ?", string(hashedPassword), user.Id).Error; dbErr != nil {
		t.Fatalf("failed to set hashed password: %v", dbErr)
	}
	user.Password = string(hashedPassword)
	return user
}

func TestUpdateUserSelfSuccess(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_self@ajuda.dev", domain.UserRoleUser)
	before := userUpdatedAt(t, user.Id)

	resp := doPutUser(t, app, user.Id, []byte(`{"name": "Lucas Rocha"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update do próprio usuário, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Name != "Lucas Rocha" {
		t.Errorf("esperava name 'Lucas Rocha' na resposta, recebeu '%s'", respDto.Name)
	}
	if respDto.Id != user.Id {
		t.Errorf("esperava id '%s' na resposta, recebeu '%s'", user.Id, respDto.Id)
	}
	if respDto.Token != "" {
		t.Errorf("esperava token vazio na resposta do update, recebeu '%s'", respDto.Token)
	}

	persisted, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after update: %v", findErr)
	}
	if persisted.Name != "Lucas Rocha" {
		t.Errorf("esperava name 'Lucas Rocha' persistido, recebeu '%s'", persisted.Name)
	}
	if after := userUpdatedAt(t, user.Id); !after.After(before) {
		t.Errorf("esperava updated_at avançado após o update (antes %v, depois %v)", before, after)
	}
}

func TestUpdateUserDoesNotChangeEmailNorPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithHashedPassword(t, app, "upd_side@ajuda.dev", domain.UserRoleUser, "123456")
	before, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user before update: %v", findErr)
	}

	resp := doPutUser(t, app, user.Id, []byte(`{"name": "Nome Novo"}`), validTokenFor(t, user.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update do próprio usuário, recebeu %d", resp.StatusCode)
	}

	after, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after update: %v", findErr)
	}
	if after.Email != before.Email {
		t.Errorf("esperava email '%s' intacto, recebeu '%s'", before.Email, after.Email)
	}
	if after.Password != before.Password {
		t.Errorf("esperava hash de senha intacto, recebeu '%s'", after.Password)
	}
	if after.Role != before.Role {
		t.Errorf("esperava role '%s' intacta, recebeu '%s'", before.Role, after.Role)
	}
}

func TestUpdateUserOtherUserForbidden(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	requester := createUserWithRole(t, "upd_other_requester@ajuda.dev", domain.UserRoleUser)
	target := createUserWithRole(t, "upd_other_target@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, target.Id, []byte(`{"name": "Invadido"}`), validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no update de outro usuário por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can update this user" {
		t.Errorf("esperava message 'only the user themselves or an admin can update this user', recebeu '%s'", respBody.Message)
	}

	persisted, findErr := userRepository.FindById(target.Id)
	if findErr != nil {
		t.Fatalf("failed to find target after forbidden update: %v", findErr)
	}
	if persisted.Name != target.Name {
		t.Errorf("esperava nome do alvo '%s' preservado após 403, recebeu '%s'", target.Name, persisted.Name)
	}
}

func TestUpdateUserModeratorCannotUpdateOthers(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	moderator := createUserWithRole(t, "upd_mod@ajuda.dev", domain.UserRoleModerator)
	target := createUserWithRole(t, "upd_mod_target@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, target.Id, []byte(`{"name": "Moderado"}`), validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no update de terceiro por MODERATOR, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can update this user" {
		t.Errorf("esperava message 'only the user themselves or an admin can update this user', recebeu '%s'", respBody.Message)
	}

	persisted, findErr := userRepository.FindById(target.Id)
	if findErr != nil {
		t.Fatalf("failed to find target after forbidden update: %v", findErr)
	}
	if persisted.Name != target.Name {
		t.Errorf("esperava nome do alvo '%s' preservado após 403 do MODERATOR, recebeu '%s'", target.Name, persisted.Name)
	}
}

func TestUpdateUserAdminCanUpdateOthers(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "upd_admin@ajuda.dev", domain.UserRoleAdmin)
	target := createUserWithRole(t, "upd_admin_target@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, target.Id, []byte(`{"name": "Alterado pelo Admin"}`), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update de terceiro por ADMIN, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Name != "Alterado pelo Admin" {
		t.Errorf("esperava name 'Alterado pelo Admin' na resposta, recebeu '%s'", respDto.Name)
	}

	persisted, findErr := userRepository.FindById(target.Id)
	if findErr != nil {
		t.Fatalf("failed to find target after admin update: %v", findErr)
	}
	if persisted.Name != "Alterado pelo Admin" {
		t.Errorf("esperava name 'Alterado pelo Admin' persistido, recebeu '%s'", persisted.Name)
	}
}

func TestUpdateUserAdminCanUpdateSelf(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "upd_admin_self@ajuda.dev", domain.UserRoleAdmin)

	resp := doPutUser(t, app, admin.Id, []byte(`{"name": "Admin Renomeado"}`), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update do próprio ADMIN, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Name != "Admin Renomeado" {
		t.Errorf("esperava name 'Admin Renomeado' na resposta, recebeu '%s'", respDto.Name)
	}
}

func TestUpdateUserUnauthorized(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_unauth@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"name": "Sem Token"}`), "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no update sem token, recebeu %d", resp.StatusCode)
	}

	resp = doPutUser(t, app, user.Id, []byte(`{"name": "Token Fantasma"}`), validTokenFor(t, uuidv7.New().String()))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no update com usuário do token inexistente no banco, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func TestUpdateUserInvalidId(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_badid@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, "abc", []byte(`{"name": "Nome"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no update com id inválido, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("userId", respBody.Causes)
	if len(causes) == 0 || causes[0] != "userId must be a valid UUID v7" {
		t.Errorf("esperava cause do campo 'userId', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserTargetNotFound(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "upd_404_admin@ajuda.dev", domain.UserRoleAdmin)

	resp := doPutUser(t, app, uuidv7.New().String(), []byte(`{"name": "Fantasma"}`), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no update de usuário inexistente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "User not found" {
		t.Errorf("esperava message 'User not found', recebeu '%s'", respBody.Message)
	}
}

func TestUpdateUserTargetSoftDeleted(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "upd_soft_admin@ajuda.dev", domain.UserRoleAdmin)
	target := createUserWithRole(t, "upd_soft_target@ajuda.dev", domain.UserRoleUser)

	if delErr := userRepository.SoftDeleteById(target.Id); delErr != nil {
		t.Fatalf("failed to soft delete user: %v", delErr)
	}

	resp := doPutUser(t, app, target.Id, []byte(`{"name": "Arquivado"}`), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no update de usuário soft-deletado, recebeu %d", resp.StatusCode)
	}
}

func TestUpdateUserEmptyBody(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_empty@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no body vazio, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "Invalid user data" {
		t.Errorf("esperava message 'Invalid user data', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("body", respBody.Causes)
	if len(causes) == 0 || causes[0] != "provide at least one field to update" {
		t.Errorf("esperava cause 'provide at least one field to update', recebeu %+v", respBody.Causes)
	}
}

func TestUpdateUserInvalidName(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_badname@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"name": "Lucas 123"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no nome inválido, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Name is not valid" {
		t.Errorf("esperava cause 'Name is not valid', recebeu %+v", respBody.Causes)
	}

	persisted, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after invalid name: %v", findErr)
	}
	if persisted.Name != user.Name {
		t.Errorf("esperava nome '%s' preservado após 400, recebeu '%s'", user.Name, persisted.Name)
	}
}

func TestUpdateUserRejectsEmail(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_email@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"email": "novo@ajudadev.dev"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no body com email, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("body", respBody.Causes)
	if len(causes) == 0 || causes[0] != "provide at least one field to update" {
		t.Errorf("esperava cause 'provide at least one field to update', recebeu %+v", respBody.Causes)
	}

	persisted, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after rejected email: %v", findErr)
	}
	if persisted.Email != user.Email {
		t.Errorf("esperava email '%s' preservado após 400, recebeu '%s'", user.Email, persisted.Email)
	}
}

func TestUpdateUserRejectsPassword(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithHashedPassword(t, app, "upd_password@ajuda.dev", domain.UserRoleUser, "123456")
	before, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user before update: %v", findErr)
	}

	resp := doPutUser(t, app, user.Id, []byte(`{"password": "nova-senha"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no body com password, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("body", respBody.Causes)
	if len(causes) == 0 || causes[0] != "provide at least one field to update" {
		t.Errorf("esperava cause 'provide at least one field to update', recebeu %+v", respBody.Causes)
	}

	after, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after rejected password: %v", findErr)
	}
	if after.Password != before.Password {
		t.Errorf("esperava hash de senha intacto após 400, recebeu '%s'", after.Password)
	}

	loginBody := []byte(`{
		"email": "upd_password@ajuda.dev",
		"password": "123456"
	}`)
	respLogin, err := app.Test(newUserLoginRequest(loginBody))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer respLogin.Body.Close()
	if respLogin.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no login com a senha antiga, recebeu %d", respLogin.StatusCode)
	}
}

func TestUpdateUserIgnoresRole(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_role@ajuda.dev", domain.UserRoleUser)

	resp := doPutUser(t, app, user.Id, []byte(`{"name": "Quase Admin", "role": "ADMIN"}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update com role no body, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Role != domain.UserRoleUser {
		t.Errorf("esperava role 'USER' na resposta, recebeu '%s'", respDto.Role)
	}

	persisted, findErr := userRepository.FindById(user.Id)
	if findErr != nil {
		t.Fatalf("failed to find user after role attempt: %v", findErr)
	}
	if persisted.Role != domain.UserRoleUser {
		t.Errorf("esperava role 'USER' preservada no banco, recebeu '%s'", persisted.Role)
	}
	if persisted.Name != "Quase Admin" {
		t.Errorf("esperava name 'Quase Admin' persistido, recebeu '%s'", persisted.Name)
	}
}

func TestUpdateUserSameNameIsNotNotFound(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "upd_noop@ajuda.dev", domain.UserRoleUser)

	body, err := json.Marshal(map[string]string{"name": user.Name})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp := doPutUser(t, app, user.Id, body, validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update sem alteração efetiva (RowsAffected 0 não é 404), recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if respDto.Name != user.Name {
		t.Errorf("esperava name '%s' na resposta, recebeu '%s'", user.Name, respDto.Name)
	}
}
