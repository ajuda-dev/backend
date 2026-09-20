package controller_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	skilldto "github.com/ajuda-dev/backend/src/controller/skill/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newSkillRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/skill/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func doRegisterSkill(t *testing.T, app *fiber.App, name string, token string) *http.Response {
	t.Helper()
	payload, err := json.Marshal(skilldto.RegisterSkillDto{Name: name})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newSkillRegisterRequest(payload), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func registerSkillViaApiAs(t *testing.T, app *fiber.App, name string, token string) skilldto.RegisterSkillDto {
	t.Helper()
	resp := doRegisterSkill(t, app, name, token)
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201 ao registrar skill '%s', recebeu %d (body: %+v)", name, resp.StatusCode, respBody)
	}
	var respDto skilldto.RegisterSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func registerSkillViaApi(t *testing.T, app *fiber.App, name string) skilldto.RegisterSkillDto {
	t.Helper()
	user := createUserWithRole(t, fmt.Sprintf("skill_reg_%s@ajuda.dev", uuidv7.New().String()), userdomain.UserRoleUser)
	return registerSkillViaApiAs(t, app, name, validTokenFor(t, user.Id))
}

func skillReqValidation(t *testing.T, app *fiber.App, body []byte) rest_err.RestErr {
	t.Helper()
	user := createUserWithRole(t, fmt.Sprintf("skill_val_%s@ajuda.dev", uuidv7.New().String()), userdomain.UserRoleUser)
	resp, err := doAuthedRequest(app, newSkillRegisterRequest(body), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	verifyCodeError(t, respBody)
	return respBody
}

func skillCleanups() {
	cleanSkillUsersTable()
	cleanSkillsTable()
	cleanUsersTable()
}

func TestRegisterSkillNormalizesToUppercase(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	respDto := registerSkillViaApi(t, app, "  java ")

	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
	if respDto.Name != "JAVA" {
		t.Errorf("esperava nome normalizado 'JAVA', recebeu '%s'", respDto.Name)
	}
}

func TestRegisterSkillRejectsDuplicateName(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	registerSkillViaApi(t, app, "java")

	respBody := skillReqValidation(t, app, []byte(`{"name": "Java"}`))
	if respBody.Message != "Invalid skill data" {
		t.Errorf("esperava message 'Invalid skill data', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Skill already exists" {
		t.Errorf("esperava cause 'Skill already exists' para o campo 'name', recebeu %+v", respBody.Causes)
	}
}

func TestRegisterSkillRejectsInvalidNames(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	invalidNames := []string{
		"",
		strings.Repeat("A", 51),
		"GO LANG!",
		"!JAVA",
		strings.Repeat(" ", 10),
	}
	for _, name := range invalidNames {
		payload, _ := json.Marshal(skilldto.RegisterSkillDto{Name: name})
		respBody := skillReqValidation(t, app, payload)
		causes := getCauseByField("name", respBody.Causes)
		if len(causes) == 0 {
			t.Errorf("esperava cause para o campo 'name' (nome '%s'), recebeu %+v", name, respBody.Causes)
		}
	}
}

func TestRegisterSkillAllowsSpecialCharsets(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	for _, name := range []string{"c++", "c#", "R2-D2", "REST API", "ÁGUA"} {
		respDto := registerSkillViaApi(t, app, name)
		expected := strings.ToUpper(strings.TrimSpace(name))
		if respDto.Name != expected {
			t.Errorf("esperava '%s', recebeu '%s'", expected, respDto.Name)
		}
	}
}

func TestListSkillsByPrefix(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	registerSkillViaApi(t, app, "go")
	registerSkillViaApi(t, app, "golang")
	registerSkillViaApi(t, app, "java")
	registerSkillViaApi(t, app, "spring")

	listSkills := func(query string) skilldto.PageableSkillDto {
		t.Helper()
		resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/skill"+query, nil), validTokenFor(t, uuidv7.New().String()))
		if err != nil {
			t.Fatalf("erro ao executar requisição: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
		}
		var page skilldto.PageableSkillDto
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			t.Fatalf("erro ao decodificar body: %v", err)
		}
		return page
	}

	page := listSkills("?name=go")
	if len(page.Data) != 2 || page.Data[0].Name != "GO" || page.Data[1].Name != "GOLANG" {
		t.Errorf("esperava [GO, GOLANG] na busca por 'go', recebeu %+v", page.Data)
	}
	if page.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}

	page = listSkills("?name=JaVa")
	if len(page.Data) != 1 || page.Data[0].Name != "JAVA" {
		t.Errorf("esperava [JAVA] na busca por 'JaVa', recebeu %+v", page.Data)
	}

	page = listSkills("?name=zzz")
	if len(page.Data) != 0 {
		t.Errorf("esperava lista vazia na busca sem match, recebeu %+v", page.Data)
	}
}

func TestListSkillsPagination(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	for i := 0; i < 13; i++ {
		registerSkillViaApi(t, app, fmt.Sprintf("SKILL%02d", i))
	}

	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/skill?limit=10", nil), validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	var page skilldto.PageableSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 10 {
		t.Errorf("esperava 10 skills na primeira página, recebeu %d", len(page.Data))
	}
	if !page.HasNext {
		t.Errorf("esperava has_next true na primeira página")
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("GET", "/v1/skill?page=2&limit=10", nil), validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	page = skilldto.PageableSkillDto{}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 3 {
		t.Errorf("esperava 3 skills na segunda página, recebeu %d", len(page.Data))
	}
	if page.HasNext {
		t.Errorf("esperava has_next false na segunda página")
	}
}

func TestUpdateSkillRenamesNormalized(t *testing.T) {
	t.Cleanup(skillCleanups)

	app := setupApp()
	moderator := createUserWithRole(t, "upd_skill_mod@ajuda.dev", userdomain.UserRoleModerator)
	skill := registerSkillViaApi(t, app, "java")

	payload := []byte(`{"name": " kotlin "}`)
	req := httptest.NewRequest("PUT", "/v1/skill/"+skill.Id, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doAuthedRequest(app, req, validTokenFor(t, moderator.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var respDto skilldto.RegisterSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.Id != skill.Id || respDto.Name != "KOTLIN" {
		t.Errorf("esperava skill renomeada para KOTLIN, recebeu %+v", respDto)
	}
}

func TestUpdateSkillRejectsConflictAndNotFound(t *testing.T) {
	t.Cleanup(skillCleanups)

	app := setupApp()
	moderator := createUserWithRole(t, "upd_skill_conflict_mod@ajuda.dev", userdomain.UserRoleModerator)
	java := registerSkillViaApi(t, app, "java")
	registerSkillViaApi(t, app, "go")

	req := httptest.NewRequest("PUT", "/v1/skill/"+java.Id, bytes.NewBuffer([]byte(`{"name": "go"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doAuthedRequest(app, req, validTokenFor(t, moderator.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no conflito de nome, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	causes := getCauseByField("name", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Skill already exists" {
		t.Errorf("esperava cause 'Skill already exists', recebeu %+v", respBody.Causes)
	}

	req = httptest.NewRequest("PUT", "/v1/skill/"+uuidv7.New().String(), bytes.NewBuffer([]byte(`{"name": "rust"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = doAuthedRequest(app, req, validTokenFor(t, moderator.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 ao renomear skill inexistente, recebeu %d", resp.StatusCode)
	}
}

func TestDeleteSkillRemovesAssociationsAndAllowsNameReuse(t *testing.T) {
	t.Cleanup(skillCleanups)

	app := setupApp()
	skill := registerSkillViaApi(t, app, "go")
	user := createSkillUserForTest(t)
	assignSkillViaApi(t, app, skill.Id, user.Id, skilldomain.LevelTeach)
	moderator := createUserWithRole(t, "mod_delete_skill@ajuda.dev", userdomain.UserRoleModerator)

	resp, err := doAuthedRequest(app, httptest.NewRequest("DELETE", "/v1/skill/"+skill.Id, nil), validTokenFor(t, moderator.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete da skill, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("GET", "/v1/skill/"+skill.Id, nil), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no GET da skill após delete, recebeu %d", resp.StatusCode)
	}

	skills := getUserSkillsViaApi(t, app, user.Id)
	if len(skills) != 0 {
		t.Errorf("esperava perfil do usuário sem a skill deletada, recebeu %+v", skills)
	}

	registerSkillViaApi(t, app, "go")
}

func TestRegisterSkillRequiresPersistedUser(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	payload := []byte(`{"name": "GHOST"}`)

	req := newSkillRegisterRequest(payload)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, newSkillRegisterRequest(payload), validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 com token fantasma, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}

	listReq := httptest.NewRequest("GET", "/v1/skill?name=GHOST", nil)
	listResp, err := doAuthedRequest(app, listReq, validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao listar skills: %v", err)
	}
	defer listResp.Body.Close()
	var page skilldto.PageableSkillDto
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar lista: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("esperava catálogo sem GHOST após 401, recebeu %+v", page.Data)
	}
}

func TestRegisterSkillAllowsAnyAuthenticatedRole(t *testing.T) {
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createUserWithRole(t, "skill_create_user@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "skill_create_mod@ajuda.dev", userdomain.UserRoleModerator)
	admin := createUserWithRole(t, "skill_create_admin@ajuda.dev", userdomain.UserRoleAdmin)

	if got := registerSkillViaApiAs(t, app, "userlang", validTokenFor(t, user.Id)); got.Name != "USERLANG" {
		t.Errorf("esperava USERLANG criado por USER, recebeu %+v", got)
	}
	if got := registerSkillViaApiAs(t, app, "modlang", validTokenFor(t, moderator.Id)); got.Name != "MODLANG" {
		t.Errorf("esperava MODLANG criado por MODERATOR, recebeu %+v", got)
	}
	if got := registerSkillViaApiAs(t, app, "adminlang", validTokenFor(t, admin.Id)); got.Name != "ADMINLANG" {
		t.Errorf("esperava ADMINLANG criado por ADMIN, recebeu %+v", got)
	}
}
