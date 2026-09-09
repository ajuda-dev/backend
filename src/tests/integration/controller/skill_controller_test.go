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
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newSkillRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/skill/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func registerSkillViaApi(t *testing.T, app *fiber.App, name string) dto.RegisterSkillDto {
	t.Helper()
	payload, err := json.Marshal(dto.RegisterSkillDto{Name: name})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := app.Test(newSkillRegisterRequest(payload))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201 ao registrar skill '%s', recebeu %d (body: %+v)", name, resp.StatusCode, respBody)
	}
	var respDto dto.RegisterSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func skillReqValidation(t *testing.T, app *fiber.App, body []byte) rest_err.RestErr {
	t.Helper()
	resp, err := app.Test(newSkillRegisterRequest(body))
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

	app := setupApp()
	invalidNames := []string{
		"",
		strings.Repeat("A", 51),
		"GO LANG!",
		"!JAVA",
		strings.Repeat(" ", 10),
	}
	for _, name := range invalidNames {
		payload, _ := json.Marshal(dto.RegisterSkillDto{Name: name})
		respBody := skillReqValidation(t, app, payload)
		causes := getCauseByField("name", respBody.Causes)
		if len(causes) == 0 {
			t.Errorf("esperava cause para o campo 'name' (nome '%s'), recebeu %+v", name, respBody.Causes)
		}
	}
}

func TestRegisterSkillAllowsSpecialCharsets(t *testing.T) {
	t.Cleanup(cleanSkillsTable)

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

	app := setupApp()
	registerSkillViaApi(t, app, "go")
	registerSkillViaApi(t, app, "golang")
	registerSkillViaApi(t, app, "java")
	registerSkillViaApi(t, app, "spring")

	listSkills := func(query string) dto.PageableSkillDto {
		t.Helper()
		resp, err := app.Test(httptest.NewRequest("GET", "/v1/skill"+query, nil))
		if err != nil {
			t.Fatalf("erro ao executar requisição: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
		}
		var page dto.PageableSkillDto
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

	app := setupApp()
	for i := 0; i < 13; i++ {
		registerSkillViaApi(t, app, fmt.Sprintf("SKILL%02d", i))
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/skill?limit=10", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	var page dto.PageableSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 10 {
		t.Errorf("esperava 10 skills na primeira página, recebeu %d", len(page.Data))
	}
	if !page.HasNext {
		t.Errorf("esperava has_next true na primeira página")
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/v1/skill?page=2&limit=10", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	page = dto.PageableSkillDto{}
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
	t.Cleanup(cleanSkillsTable)

	app := setupApp()
	skill := registerSkillViaApi(t, app, "java")

	payload := []byte(`{"name": " kotlin "}`)
	req := httptest.NewRequest("PUT", "/v1/skill/"+skill.Id, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var respDto dto.RegisterSkillDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.Id != skill.Id || respDto.Name != "KOTLIN" {
		t.Errorf("esperava skill renomeada para KOTLIN, recebeu %+v", respDto)
	}
}

func TestUpdateSkillRejectsConflictAndNotFound(t *testing.T) {
	t.Cleanup(cleanSkillsTable)

	app := setupApp()
	java := registerSkillViaApi(t, app, "java")
	registerSkillViaApi(t, app, "go")

	req := httptest.NewRequest("PUT", "/v1/skill/"+java.Id, bytes.NewBuffer([]byte(`{"name": "go"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
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
	resp, err = app.Test(req)
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
	assignSkillViaApi(t, app, skill.Id, user.Id, domain.LevelTeach)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/v1/skill/"+skill.Id, nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no delete da skill, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/v1/skill/"+skill.Id, nil))
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
