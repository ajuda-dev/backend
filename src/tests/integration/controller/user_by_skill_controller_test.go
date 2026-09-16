package controller_test

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func createUserForSkillSearch(t *testing.T, name string, email string) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     name,
		Email:    email,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user '%s': %v", name, createErr)
	}
	return user
}

func searchUsersQuery(t *testing.T, app *fiber.App, query string) userdto.PageableUserDto {
	t.Helper()
	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/user"+query, nil), validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na busca de usuários, recebeu %d", resp.StatusCode)
	}
	var page userdto.PageableUserDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func userSearchIds(page userdto.PageableUserDto) []string {
	ids := make([]string, len(page.Data))
	for i, u := range page.Data {
		ids[i] = u.Id
	}
	return ids
}

func TestGetUsersBySkillReturnsMatchingUsersWithProfile(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	lucas := createUserForSkillSearch(t, "lucas.darocha", "lucas@ajuda.dev")
	maria := createUserForSkillSearch(t, "maria", "maria@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	goSkill := createSkillForTest(t, "GO")
	spring := createSkillForTest(t, "SPRING")
	assignSkillViaApi(t, app, java.Id, lucas.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, goSkill.Id, lucas.Id, skilldomain.LevelLearnAndTeach)
	assignSkillViaApi(t, app, spring.Id, maria.Id, skilldomain.LevelWantToLearn)

	page := searchUsersQuery(t, app, "?skill=JAVA")
	if page.HasNext {
		t.Error("esperava has_next false, recebeu true")
	}
	if len(page.Data) != 1 {
		t.Fatalf("esperava somente o usuário com JAVA, recebeu %d usuários", len(page.Data))
	}
	user := page.Data[0]
	if user.Id != lucas.Id {
		t.Errorf("esperava id '%s', recebeu '%s'", lucas.Id, user.Id)
	}
	if user.Name != "lucas.darocha" {
		t.Errorf("esperava name 'lucas.darocha', recebeu '%s'", user.Name)
	}
	skills := make([]string, len(user.Skills))
	for i, s := range user.Skills {
		skills[i] = s.Name
	}
	if len(user.Skills) != 2 || skills[0] != "GO" || skills[1] != "JAVA" {
		t.Errorf("esperava skills do perfil [GO, JAVA] em ordem alfabética, recebeu %+v", skills)
	}
}

func TestGetUsersBySkillNormalizesCase(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	lucas := createUserForSkillSearch(t, "lucas.darocha", "lucas@ajuda.dev")
	maria := createUserForSkillSearch(t, "maria", "maria@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, java.Id, lucas.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, java.Id, maria.Id, skilldomain.LevelWantToLearn)

	for _, query := range []string{
		"?skill=JAVA",
		"?skill=java",
		"?skill=" + url.QueryEscape(" JaVa "),
	} {
		page := searchUsersQuery(t, app, query)
		if len(page.Data) != 2 {
			t.Fatalf("busca '%s': esperava 2 usuários, recebeu %d", query, len(page.Data))
		}
		ids := userSearchIds(page)
		if ids[0] != lucas.Id || ids[1] != maria.Id {
			t.Errorf("busca '%s': esperava [lucas.darocha, maria] ordenados por nome, recebeu %+v", query, ids)
		}
	}
}

func TestGetUsersBySkillNonexistentSkillReturnsEmpty(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	lucas := createUserForSkillSearch(t, "lucas", "lucas@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, java.Id, lucas.Id, skilldomain.LevelTeach)

	page := searchUsersQuery(t, app, "?skill=RUST")
	if len(page.Data) != 0 {
		t.Errorf("esperava data vazio para skill sem donos, recebeu %+v", page.Data)
	}
	if page.HasNext {
		t.Error("esperava has_next false, recebeu true")
	}
}

func TestGetUsersBySkillLengthValidation(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()

	target := "/v1/user?skill=" + strings.Repeat("A", 51)
	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", target, nil), validTokenFor(t, uuidv7.New().String()))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para '%s', recebeu %d", target, resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	verifyCodeError(t, respBody)
	causes := getCauseByField("skill", respBody.Causes)
	if len(causes) == 0 || causes[0] != "Skill name is not valid" {
		t.Errorf("esperava cause 'Skill name is not valid' para o campo 'skill', recebeu %+v", respBody.Causes)
	}
}

func TestGetUsersBySkillExactMatch(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	lucas := createUserForSkillSearch(t, "lucas", "lucas@ajuda.dev")
	maria := createUserForSkillSearch(t, "maria", "maria@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	javaScript := createSkillForTest(t, "JAVASCRIPT")
	assignSkillViaApi(t, app, java.Id, lucas.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, javaScript.Id, maria.Id, skilldomain.LevelWantToLearn)

	page := searchUsersQuery(t, app, "?skill=JAVA")
	ids := userSearchIds(page)
	if len(ids) != 1 || ids[0] != lucas.Id {
		t.Errorf("busca por JAVA não deve retornar quem tem JAVASCRIPT; recebeu %+v", page.Data)
	}

	page = searchUsersQuery(t, app, "?skill=JAVASCRIPT")
	ids = userSearchIds(page)
	if len(ids) != 1 || ids[0] != maria.Id {
		t.Errorf("busca por JAVASCRIPT deve retornar só a maria; recebeu %+v", page.Data)
	}
}

func TestGetUsersBySkillAfterSkillSoftDelete(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	lucas := createUserForSkillSearch(t, "lucas", "lucas@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, java.Id, lucas.Id, skilldomain.LevelTeach)
	moderator := createUserWithRole(t, "user_skill_search_mod@ajuda.dev", userdomain.UserRoleModerator)

	resp, err := doAuthedRequest(app, httptest.NewRequest("DELETE", "/v1/skill/"+java.Id, nil), validTokenFor(t, moderator.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 no soft delete da skill, recebeu %d", resp.StatusCode)
	}

	page := searchUsersQuery(t, app, "?skill=JAVA")
	if len(page.Data) != 0 {
		t.Errorf("esperava ninguém após soft delete da skill, recebeu %+v", page.Data)
	}
	skills := getUserSkillsViaApi(t, app, lucas.Id)
	if len(skills) != 0 {
		t.Errorf("esperava perfil do usuário sem a skill deletada, recebeu %+v", skills)
	}
}

func TestGetUsersBySkillPagination(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	alpha := createUserForSkillSearch(t, "alpha", "alpha@ajuda.dev")
	bravo := createUserForSkillSearch(t, "bravo", "bravo@ajuda.dev")
	charlie := createUserForSkillSearch(t, "charlie", "charlie@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	goSkill := createSkillForTest(t, "GO")
	spring := createSkillForTest(t, "SPRING")
	assignSkillViaApi(t, app, java.Id, alpha.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, java.Id, bravo.Id, skilldomain.LevelLearnAndTeach)
	assignSkillViaApi(t, app, java.Id, charlie.Id, skilldomain.LevelWantToLearn)
	assignSkillViaApi(t, app, goSkill.Id, alpha.Id, skilldomain.LevelLearnAndTeach)
	assignSkillViaApi(t, app, spring.Id, bravo.Id, skilldomain.LevelTeach)

	skillNames := func(page userdto.PageableUserDto, index int) []string {
		names := make([]string, len(page.Data[index].Skills))
		for i, s := range page.Data[index].Skills {
			names[i] = s.Name
		}
		return names
	}

	firstPage := searchUsersQuery(t, app, "?skill=JAVA&limit=2")
	if len(firstPage.Data) != 2 {
		t.Fatalf("esperava 2 usuários na primeira página, recebeu %d", len(firstPage.Data))
	}
	if !firstPage.HasNext {
		t.Error("esperava has_next true na primeira página")
	}
	firstIds := userSearchIds(firstPage)
	if firstIds[0] != alpha.Id || firstIds[1] != bravo.Id {
		t.Errorf("esperava [alpha, bravo] na primeira página, recebeu %+v", firstIds)
	}
	alphaSkills := skillNames(firstPage, 0)
	if len(alphaSkills) != 2 || alphaSkills[0] != "GO" || alphaSkills[1] != "JAVA" {
		t.Errorf("esperava skills de alpha [GO, JAVA], recebeu %+v", alphaSkills)
	}
	bravoSkills := skillNames(firstPage, 1)
	if len(bravoSkills) != 2 || bravoSkills[0] != "JAVA" || bravoSkills[1] != "SPRING" {
		t.Errorf("esperava skills de bravo [JAVA, SPRING], recebeu %+v", bravoSkills)
	}

	secondPage := searchUsersQuery(t, app, "?skill=JAVA&page=2&limit=2")
	if len(secondPage.Data) != 1 {
		t.Fatalf("esperava 1 usuário na segunda página, recebeu %d", len(secondPage.Data))
	}
	if secondPage.HasNext {
		t.Error("esperava has_next false na segunda página")
	}
	secondIds := userSearchIds(secondPage)
	if secondIds[0] != charlie.Id {
		t.Errorf("esperava [charlie] na segunda página, recebeu %+v", secondIds)
	}
	charlieSkills := skillNames(secondPage, 0)
	if len(charlieSkills) != 1 || charlieSkills[0] != "JAVA" {
		t.Errorf("esperava skills de charlie [JAVA], recebeu %+v", charlieSkills)
	}
}
