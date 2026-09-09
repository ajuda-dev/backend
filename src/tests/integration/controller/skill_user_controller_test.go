package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func newAssignSkillRequest(skillId string, body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/skill/"+skillId+"/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createSkillUserForTest(t *testing.T) *domain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "lucas",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

func createSkillForTest(t *testing.T, name string) *domain.SkillDomain {
	t.Helper()
	skill, createErr := skillRepository.CreateSkill(&domain.SkillDomain{Name: name})
	if createErr != nil {
		t.Fatalf("failed to create skill: %v", createErr)
	}
	return skill
}

func assignSkillViaApi(t *testing.T, app *fiber.App, skillId string, userId string, level string) dto.SkillUserDto {
	t.Helper()
	payload, err := json.Marshal(dto.AssignSkillDto{UserId: userId, Level: level})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := app.Test(newAssignSkillRequest(skillId, payload))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201 ao associar skill, recebeu %d (body: %+v)", resp.StatusCode, respBody)
	}
	var respDto dto.SkillUserDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func getUserSkillsViaApi(t *testing.T, app *fiber.App, userId string) []dto.SkillUserDto {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", "/v1/user/"+userId+"/skills", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var skills []dto.SkillUserDto
	if err := json.NewDecoder(resp.Body).Decode(&skills); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return skills
}

func skillUserNames(skills []dto.SkillUserDto) []string {
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

	skillUser := assignSkillViaApi(t, app, skill.Id, user.Id, domain.LevelWantToLearn)

	if !uuidv7.IsValidString(skillUser.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", skillUser.Id)
	}
	if skillUser.SkillId != skill.Id {
		t.Errorf("esperava skill_id '%s', recebeu '%s'", skill.Id, skillUser.SkillId)
	}
	if skillUser.UserId != user.Id {
		t.Errorf("esperava user_id '%s', recebeu '%s'", user.Id, skillUser.UserId)
	}
	if skillUser.Level != domain.LevelWantToLearn {
		t.Errorf("esperava level '%s', recebeu '%s'", domain.LevelWantToLearn, skillUser.Level)
	}
}

func TestAssignSkillRejectsDuplicate(t *testing.T) {
	t.Cleanup(cleanSkillUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	user := createSkillUserForTest(t)
	skill := createSkillForTest(t, "GO")
	assignSkillViaApi(t, app, skill.Id, user.Id, domain.LevelTeach)

	payload, _ := json.Marshal(dto.AssignSkillDto{UserId: user.Id, Level: domain.LevelLearnAndTeach})
	resp, err := app.Test(newAssignSkillRequest(skill.Id, payload))
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

	for _, level := range []string{"", "SENIOR", "want_to_learn"} {
		payload, _ := json.Marshal(dto.AssignSkillDto{UserId: user.Id, Level: level})
		resp, err := app.Test(newAssignSkillRequest(skill.Id, payload))
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

	resp, err := app.Test(newAssignSkillRequest("not-a-uuid", []byte(`{"user_id": "`+user.Id+`", "level": "TEACH"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com skillId inválido, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(newAssignSkillRequest(skill.Id, []byte(`{"user_id": "not-a-uuid", "level": "TEACH"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com user_id inválido, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(newAssignSkillRequest(uuidv7.New().String(), []byte(`{"user_id": "`+user.Id+`", "level": "TEACH"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com skill inexistente, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(newAssignSkillRequest(skill.Id, []byte(`{"user_id": "`+uuidv7.New().String()+`", "level": "TEACH"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com usuário inexistente, recebeu %d", resp.StatusCode)
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

	assignSkillViaApi(t, app, java.Id, user.Id, domain.LevelWantToLearn)
	assignSkillViaApi(t, app, spring.Id, user.Id, domain.LevelLearnAndTeach)
	assignSkillViaApi(t, app, goSkill.Id, user.Id, domain.LevelTeach)

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
	if levelsBySkill["GO"] != domain.LevelTeach || levelsBySkill["JAVA"] != domain.LevelWantToLearn || levelsBySkill["SPRING"] != domain.LevelLearnAndTeach {
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
	assignSkillViaApi(t, app, java.Id, user.Id, domain.LevelWantToLearn)
	assignSkillViaApi(t, app, goSkill.Id, user.Id, domain.LevelTeach)

	req := httptest.NewRequest("DELETE", "/v1/user/"+user.Id+"/skills/"+java.Id, nil)
	resp, err := app.Test(req)
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

	resp, err = app.Test(httptest.NewRequest("DELETE", "/v1/user/"+user.Id+"/skills/"+java.Id, nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 ao remover associação inexistente, recebeu %d", resp.StatusCode)
	}
}
