package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	skilldto "github.com/ajuda-dev/backend/src/controller/skill/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newAssignSkillRequest(skillId string, body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/skill/"+skillId+"/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assignSkillBody(t *testing.T, userId string, level string) []byte {
	t.Helper()
	payload, err := json.Marshal(skilldto.AssignSkillDto{UserId: userId, Level: level})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	return payload
}

func doAssignSkill(t *testing.T, app *fiber.App, skillId string, body []byte, token string) *http.Response {
	t.Helper()
	req := newAssignSkillRequest(skillId, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func doRemoveSkill(t *testing.T, app *fiber.App, userId string, skillId string, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("DELETE", "/v1/user/"+userId+"/skills/"+skillId, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func createSkillUserForTest(t *testing.T) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "lucas",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

func createSkillForTest(t *testing.T, name string) *skilldomain.SkillDomain {
	t.Helper()
	skill, createErr := skillRepository.CreateSkill(&skilldomain.SkillDomain{Name: name})
	if createErr != nil {
		t.Fatalf("failed to create skill: %v", createErr)
	}
	return skill
}

func assignSkillViaApi(t *testing.T, app *fiber.App, skillId string, userId string, level string) skilldto.SkillUserDto {
	t.Helper()
	payload, err := json.Marshal(skilldto.AssignSkillDto{UserId: userId, Level: level})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newAssignSkillRequest(skillId, payload), validTokenFor(t, userId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201 ao associar skill, recebeu %d (body: %+v)", resp.StatusCode, respBody)
	}
	var respDto skilldto.SkillUserDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func getUserSkillsViaApi(t *testing.T, app *fiber.App, userId string) []skilldto.SkillUserDto {
	t.Helper()
	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/user/"+userId+"/skills", nil), validTokenFor(t, userId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var skills []skilldto.SkillUserDto
	if err := json.NewDecoder(resp.Body).Decode(&skills); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return skills
}

func skillUserNames(skills []skilldto.SkillUserDto) []string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Skill.Name
	}
	return names
}

func TestAssignSkillSuccess(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	skill := createSkillForTest(t, "JAVA")

	skillUser := assignSkillViaApi(t, app, skill.Id, user.Id, skilldomain.LevelWantToLearn)

	if !uuidv7.IsValidString(skillUser.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", skillUser.Id)
	}
	if skillUser.SkillId != skill.Id {
		t.Errorf("esperava skill_id '%s', recebeu '%s'", skill.Id, skillUser.SkillId)
	}
	if skillUser.UserId != user.Id {
		t.Errorf("esperava user_id '%s', recebeu '%s'", user.Id, skillUser.UserId)
	}
	if skillUser.Level != skilldomain.LevelWantToLearn {
		t.Errorf("esperava level '%s', recebeu '%s'", skilldomain.LevelWantToLearn, skillUser.Level)
	}
}

func TestAssignSkillRejectsDuplicate(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	skill := createSkillForTest(t, "GO")
	assignSkillViaApi(t, app, skill.Id, user.Id, skilldomain.LevelTeach)

	payload, _ := json.Marshal(skilldto.AssignSkillDto{UserId: user.Id, Level: skilldomain.LevelLearnAndTeach})
	resp, err := doAuthedRequest(app, newAssignSkillRequest(skill.Id, payload), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no assign duplicado, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	causes := getCauseByField("skill_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "user already has this skill" {
		t.Errorf("esperava cause 'user already has this skill', recebeu %+v", respBody.Causes)
	}
}

func TestAssignSkillRejectsInvalidLevel(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	skill := createSkillForTest(t, "JAVA")

	token := validTokenFor(t, user.Id)
	for _, level := range []string{"", "SENIOR", "want_to_learn"} {
		payload, _ := json.Marshal(skilldto.AssignSkillDto{UserId: user.Id, Level: level})
		resp, err := doAuthedRequest(app, newAssignSkillRequest(skill.Id, payload), token)
		if err != nil {
			t.Fatalf("erro ao executar requisição: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("esperava 400 para level '%s', recebeu %d", level, resp.StatusCode)
		}
		var respBody rest_err.RestErr
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("erro ao decodificar body: %v", err)
		}
		causes := getCauseByField("level", respBody.Causes)
		if len(causes) == 0 {
			t.Errorf("esperava cause para o campo 'level', recebeu %+v", respBody.Causes)
		}
	}
}

func TestAssignSkillValidationFailures(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	skill := createSkillForTest(t, "JAVA")

	token := validTokenFor(t, user.Id)

	resp, err := doAuthedRequest(app, newAssignSkillRequest("not-a-uuid", []byte(`{"user_id": "`+user.Id+`", "level": "TEACH"}`)), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com skillId inválido, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, newAssignSkillRequest(skill.Id, []byte(`{"user_id": "not-a-uuid", "level": "TEACH"}`)), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com user_id inválido, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, newAssignSkillRequest(uuidv7.New().String(), []byte(`{"user_id": "`+user.Id+`", "level": "TEACH"}`)), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com skill inexistente, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, newAssignSkillRequest(skill.Id, []byte(`{"user_id": "`+uuidv7.New().String()+`", "level": "TEACH"}`)), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 com usuário inexistente e token de USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}
}

func TestGetUserSkillsSortedAlphabetically(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	java := createSkillForTest(t, "JAVA")
	spring := createSkillForTest(t, "SPRING")
	goSkill := createSkillForTest(t, "GO")

	assignSkillViaApi(t, app, java.Id, user.Id, skilldomain.LevelWantToLearn)
	assignSkillViaApi(t, app, spring.Id, user.Id, skilldomain.LevelLearnAndTeach)
	assignSkillViaApi(t, app, goSkill.Id, user.Id, skilldomain.LevelTeach)

	skills := getUserSkillsViaApi(t, app, user.Id)
	names := skillUserNames(skills)
	expected := []string{"GO", "JAVA", "SPRING"}
	if len(skills) != 3 {
		t.Fatalf("esperava 3 skills, recebeu %d", len(skills))
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("posição %d: esperava '%s', recebeu '%s' (lista %+v)", i, name, names[i], names)
		}
	}
	levelsBySkill := map[string]string{}
	for _, s := range skills {
		levelsBySkill[s.Skill.Name] = s.Level
	}
	if levelsBySkill["GO"] != skilldomain.LevelTeach || levelsBySkill["JAVA"] != skilldomain.LevelWantToLearn || levelsBySkill["SPRING"] != skilldomain.LevelLearnAndTeach {
		t.Errorf("esperava níveis preservados, recebeu %+v", levelsBySkill)
	}

	empty := getUserSkillsViaApi(t, app, uuidv7.New().String())
	if len(empty) != 0 {
		t.Errorf("esperava lista vazia para usuário inexistente, recebeu %+v", empty)
	}
}

func TestRemoveSkillFromUser(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	java := createSkillForTest(t, "JAVA")
	goSkill := createSkillForTest(t, "GO")
	assignSkillViaApi(t, app, java.Id, user.Id, skilldomain.LevelWantToLearn)
	assignSkillViaApi(t, app, goSkill.Id, user.Id, skilldomain.LevelTeach)

	token := validTokenFor(t, user.Id)
	req := httptest.NewRequest("DELETE", "/v1/user/"+user.Id+"/skills/"+java.Id, nil)
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 ao remover skill do usuário, recebeu %d", resp.StatusCode)
	}

	skills := getUserSkillsViaApi(t, app, user.Id)
	if len(skills) != 1 || skills[0].Skill.Name != "GO" {
		t.Errorf("esperava somente [GO] após remoção, recebeu %+v", skills)
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("DELETE", "/v1/user/"+user.Id+"/skills/"+java.Id, nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 ao remover associação inexistente, recebeu %d", resp.StatusCode)
	}
}

func TestAssignSkillOtherUserForbidden(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	requester := createUserWithRole(t, "skill_own_req@ajuda.dev", userdomain.UserRoleUser)
	target := createUserWithRole(t, "skill_own_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")

	resp := doAssignSkill(t, app, skill.Id, assignSkillBody(t, target.Id, skilldomain.LevelWantToLearn), validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no assign de outro usuário por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 0 {
		t.Errorf("esperava alvo sem a skill após 403, recebeu %+v", skills)
	}
}

func TestAssignSkillModeratorCannotAssignToOthers(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	moderator := createUserWithRole(t, "skill_own_mod@ajuda.dev", userdomain.UserRoleModerator)
	target := createUserWithRole(t, "skill_own_mod_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "GO")

	resp := doAssignSkill(t, app, skill.Id, assignSkillBody(t, target.Id, skilldomain.LevelTeach), validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no assign de terceiro por MODERATOR, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 0 {
		t.Errorf("esperava alvo sem a skill após 403 do MODERATOR, recebeu %+v", skills)
	}
}

func TestAssignSkillAdminCanAssignToOthers(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "skill_own_admin@ajuda.dev", userdomain.UserRoleAdmin)
	target := createUserWithRole(t, "skill_own_admin_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")

	resp := doAssignSkill(t, app, skill.Id, assignSkillBody(t, target.Id, skilldomain.LevelWantToLearn), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no assign de terceiro por ADMIN, recebeu %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var respDto skilldto.SkillUserDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.UserId != target.Id {
		t.Errorf("esperava user_id do alvo '%s', recebeu '%s'", target.Id, respDto.UserId)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 1 || skills[0].Skill.Name != "JAVA" {
		t.Errorf("esperava skill JAVA no alvo após assign do ADMIN, recebeu %+v", skills)
	}
}

func TestAssignSkillAdminAndModeratorCanAssignToSelf(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "skill_own_admin_self@ajuda.dev", userdomain.UserRoleAdmin)
	moderator := createUserWithRole(t, "skill_own_mod_self@ajuda.dev", userdomain.UserRoleModerator)
	java := createSkillForTest(t, "JAVA")
	goSkill := createSkillForTest(t, "GO")

	adminResp := doAssignSkill(t, app, java.Id, assignSkillBody(t, admin.Id, skilldomain.LevelTeach), validTokenFor(t, admin.Id))
	adminResp.Body.Close()
	if adminResp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no assign do ADMIN em si, recebeu %d", adminResp.StatusCode)
	}

	modResp := doAssignSkill(t, app, goSkill.Id, assignSkillBody(t, moderator.Id, skilldomain.LevelLearnAndTeach), validTokenFor(t, moderator.Id))
	modResp.Body.Close()
	if modResp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no assign do MODERATOR em si, recebeu %d", modResp.StatusCode)
	}
}

func TestAssignSkillUnauthorized(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "skill_own_unauth@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")
	body := assignSkillBody(t, user.Id, skilldomain.LevelWantToLearn)

	resp := doAssignSkill(t, app, skill.Id, body, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no assign sem token, recebeu %d", resp.StatusCode)
	}

	resp = doAssignSkill(t, app, skill.Id, body, validTokenFor(t, uuidv7.New().String()))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no assign com token de usuário inexistente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func TestAssignSkillAdminTargetNotFound(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "skill_own_admin_404@ajuda.dev", userdomain.UserRoleAdmin)
	skill := createSkillForTest(t, "JAVA")

	resp := doAssignSkill(t, app, skill.Id, assignSkillBody(t, uuidv7.New().String(), skilldomain.LevelWantToLearn), validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no assign de usuário inexistente por ADMIN, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("user_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "user_id is not valid, not found this user" {
		t.Errorf("esperava cause 'user_id is not valid, not found this user', recebeu %+v", respBody.Causes)
	}
}

func TestRemoveSkillFromOtherUserForbidden(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	requester := createUserWithRole(t, "skill_rm_req@ajuda.dev", userdomain.UserRoleUser)
	target := createUserWithRole(t, "skill_rm_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, skill.Id, target.Id, skilldomain.LevelWantToLearn)

	resp := doRemoveSkill(t, app, target.Id, skill.Id, validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no remove de outro usuário por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 1 || skills[0].Skill.Name != "JAVA" {
		t.Errorf("esperava associação JAVA preservada após 403, recebeu %+v", skills)
	}
}

func TestRemoveSkillModeratorCannotRemoveFromOthers(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	moderator := createUserWithRole(t, "skill_rm_mod@ajuda.dev", userdomain.UserRoleModerator)
	target := createUserWithRole(t, "skill_rm_mod_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "GO")
	assignSkillViaApi(t, app, skill.Id, target.Id, skilldomain.LevelTeach)

	resp := doRemoveSkill(t, app, target.Id, skill.Id, validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no remove de terceiro por MODERATOR, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 1 || skills[0].Skill.Name != "GO" {
		t.Errorf("esperava associação GO preservada após 403 do MODERATOR, recebeu %+v", skills)
	}
}

func TestRemoveSkillAdminCanRemoveFromOthers(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "skill_rm_admin@ajuda.dev", userdomain.UserRoleAdmin)
	target := createUserWithRole(t, "skill_rm_admin_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, skill.Id, target.Id, skilldomain.LevelWantToLearn)

	resp := doRemoveSkill(t, app, target.Id, skill.Id, validTokenFor(t, admin.Id))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no remove de terceiro por ADMIN, recebeu %d", resp.StatusCode)
	}

	skills := getUserSkillsViaApi(t, app, target.Id)
	if len(skills) != 0 {
		t.Errorf("esperava alvo sem skills após remove do ADMIN, recebeu %+v", skills)
	}
}

func TestRemoveSkillUnauthorized(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "skill_rm_unauth@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, skill.Id, user.Id, skilldomain.LevelWantToLearn)

	resp := doRemoveSkill(t, app, user.Id, skill.Id, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no remove sem token, recebeu %d", resp.StatusCode)
	}

	resp = doRemoveSkill(t, app, user.Id, skill.Id, validTokenFor(t, uuidv7.New().String()))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no remove com token de usuário inexistente, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}

	skills := getUserSkillsViaApi(t, app, user.Id)
	if len(skills) != 1 || skills[0].Skill.Name != "JAVA" {
		t.Errorf("esperava associação JAVA preservada após 401, recebeu %+v", skills)
	}
}

func TestRemoveSkillUserTargetNotFoundForbidden(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "skill_rm_user_404@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")

	resp := doRemoveSkill(t, app, uuidv7.New().String(), skill.Id, validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no remove de usuário inexistente por USER, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the user themselves or an admin can manage this user's skills" {
		t.Errorf("esperava message 'only the user themselves or an admin can manage this user's skills', recebeu '%s'", respBody.Message)
	}
}

func TestRemoveSkillAdminAssociationNotFound(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	admin := createUserWithRole(t, "skill_rm_admin_404@ajuda.dev", userdomain.UserRoleAdmin)
	target := createUserWithRole(t, "skill_rm_admin_404_target@ajuda.dev", userdomain.UserRoleUser)
	skill := createSkillForTest(t, "JAVA")

	resp := doRemoveSkill(t, app, target.Id, skill.Id, validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no remove de associação inexistente por ADMIN, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "skill_user not found" {
		t.Errorf("esperava message 'skill_user not found', recebeu '%s'", respBody.Message)
	}
}
