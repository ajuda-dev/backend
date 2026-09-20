package controller_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func listUsersRaw(t *testing.T, app *fiber.App, query string, token string) (int, string) {
	t.Helper()
	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/user"+query, nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	return resp.StatusCode, string(body)
}

func listUsers(t *testing.T, app *fiber.App, query string, token string) userdto.PageableUserDto {
	t.Helper()
	status, body := listUsersRaw(t, app, query, token)
	if status != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem de usuários, recebeu %d (body: %s)", status, body)
	}
	var page userdto.PageableUserDto
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func userNames(page userdto.PageableUserDto) []string {
	names := make([]string, len(page.Data))
	for i, u := range page.Data {
		names[i] = u.Name
	}
	return names
}

func TestListUsersWithoutFiltersReturnsAllUsers(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	bruno := createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")
	carla := createUserForSkillSearch(t, "carla", "carla@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	goSkill := createSkillForTest(t, "GO")
	assignSkillViaApi(t, app, java.Id, ana.Id, skilldomain.LevelTeach)
	assignSkillViaApi(t, app, goSkill.Id, bruno.Id, skilldomain.LevelWantToLearn)

	page := listUsers(t, app, "", validTokenFor(t, ana.Id))
	if page.HasNext {
		t.Error("esperava has_next false, recebeu true")
	}
	if len(page.Data) != 3 {
		t.Fatalf("esperava os 3 usuários ativos, recebeu %d: %+v", len(page.Data), page.Data)
	}
	names := userNames(page)
	if names[0] != "ana" || names[1] != "bruno" || names[2] != "carla" {
		t.Errorf("esperava [ana, bruno, carla] ordenados por nome, recebeu %+v", names)
	}
	ids := userSearchIds(page)
	if ids[0] != ana.Id || ids[1] != bruno.Id || ids[2] != carla.Id {
		t.Errorf("esperava os ids de ana, bruno e carla, recebeu %+v", ids)
	}
	if len(page.Data[2].Skills) != 0 {
		t.Errorf("esperava skills vazio para o usuário sem skills, recebeu %+v", page.Data[2].Skills)
	}
	if len(page.Data[0].Skills) != 1 || page.Data[0].Skills[0].Name != "JAVA" {
		t.Errorf("esperava skills [JAVA] para ana, recebeu %+v", page.Data[0].Skills)
	}
}

func TestListUsersIgnoresBlankSkillFilter(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")
	token := validTokenFor(t, ana.Id)

	for _, query := range []string{"", "?skill=", "?skill=%20%20"} {
		page := listUsers(t, app, query, token)
		if len(page.Data) != 2 || page.HasNext {
			t.Errorf("query %q: esperava a lista completa (2 itens) sem filtro, recebeu %d itens, has_next=%v", query, len(page.Data), page.HasNext)
		}
	}
}

func TestListUsersExcludesSoftDeletedUsers(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	bruno := createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")
	if err := userRepository.SoftDeleteById(bruno.Id); err != nil {
		t.Fatalf("erro ao soft deletar usuário: %v", err)
	}

	page := listUsers(t, app, "", validTokenFor(t, ana.Id))
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava somente a ana (bruno soft-deletado), recebeu %+v", page.Data)
	}
}

func TestListUsersByNameSubstring(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	mariana := createUserForSkillSearch(t, "mariana", "mariana@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")

	page := listUsers(t, app, "?name=an", validTokenFor(t, ana.Id))
	if len(page.Data) != 2 {
		t.Fatalf("esperava 2 usuários contendo 'an', recebeu %d: %+v", len(page.Data), page.Data)
	}
	ids := userSearchIds(page)
	if ids[0] != ana.Id || ids[1] != mariana.Id {
		t.Errorf("esperava [ana, mariana] ordenados por nome, recebeu %+v", ids)
	}
}

func TestListUsersByNameCaseInsensitive(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana silva", "ana.silva@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")

	token := validTokenFor(t, ana.Id)
	for _, query := range []string{"?name=ANA", "?name=ana%20SILVA", "?name=%20ana"} {
		page := listUsers(t, app, query, token)
		if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
			t.Errorf("query %q: esperava somente a ana, recebeu %+v", query, page.Data)
		}
	}
}

func TestListUsersByNameAccentInsensitive(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	jose := createUserForSkillSearch(t, "José", "jose@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")

	token := validTokenFor(t, jose.Id)
	for _, query := range []string{"?name=jose", "?name=JOSÉ", "?name=josé"} {
		page := listUsers(t, app, query, token)
		if len(page.Data) != 1 || page.Data[0].Id != jose.Id {
			t.Errorf("query %q: esperava somente o José, recebeu %+v", query, page.Data)
		}
	}
}

func TestListUsersByNameAccentInsensitiveReverse(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	jose := createUserForSkillSearch(t, "Jose", "jose@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")

	page := listUsers(t, app, "?name=josé", validTokenFor(t, jose.Id))
	if len(page.Data) != 1 || page.Data[0].Id != jose.Id {
		t.Errorf("esperava o usuário 'Jose' para termo acentuado, recebeu %+v", page.Data)
	}
}

func TestListUsersByNameEscape(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana silva", "ana.silva@ajuda.dev")
	token := validTokenFor(t, ana.Id)

	wildcardPage := listUsers(t, app, "?name=%25", token)
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listUsers(t, app, "?name=ana_silva", token)
	if len(underscorePage.Data) != 0 {
		t.Errorf("esperava 0 resultados para 'ana_silva' (underscore literal), recebeu %+v", underscorePage.Data)
	}
}

func TestListUsersByNameAndEmailAndSkill(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana silva", "ana.silva@ajuda.dev")
	createUserForSkillSearch(t, "ana souza", "ana.souza@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, java.Id, ana.Id, skilldomain.LevelTeach)
	admin := createUserWithRole(t, "list_combo_admin@ajuda.dev", userdomain.UserRoleAdmin)
	adminToken := validTokenFor(t, admin.Id)

	page := listUsers(t, app, "?skill=JAVA&name=ana&email=ana.silva@ajuda.dev", adminToken)
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava somente a ana com os 3 filtros combinados, recebeu %+v", page.Data)
	}

	noMatch := listUsers(t, app, "?skill=JAVA&name=ana&email=ana.souza@ajuda.dev", adminToken)
	if len(noMatch.Data) != 0 {
		t.Errorf("esperava vazio quando um dos filtros não casa, recebeu %+v", noMatch.Data)
	}
}

func TestListUsersByEmailExactMatch(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana silva", "ana.silva@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")
	admin := createUserWithRole(t, "list_email_admin@ajuda.dev", userdomain.UserRoleAdmin)
	adminToken := validTokenFor(t, admin.Id)

	page := listUsers(t, app, "?email=ANA.SILVA@AJUDA.DEV", adminToken)
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava a ana no match exato case-insensitive, recebeu %+v", page.Data)
	}

	for _, query := range []string{"?email=ana.silva", "?email=ajuda.dev", "?email=nao.existe@ajuda.dev"} {
		partial := listUsers(t, app, query, adminToken)
		if len(partial.Data) != 0 {
			t.Errorf("query %q: esperava vazio (match exato), recebeu %+v", query, partial.Data)
		}
	}

	blank := listUsers(t, app, "?email=", validTokenFor(t, ana.Id))
	if len(blank.Data) != 3 {
		t.Errorf("esperava a lista completa com email vazio, recebeu %+v", blank.Data)
	}
}

func TestListUsersPaginationWithoutSkill(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	alpha := createUserForSkillSearch(t, "alpha", "alpha@ajuda.dev")
	bravo := createUserForSkillSearch(t, "bravo", "bravo@ajuda.dev")
	charlie := createUserForSkillSearch(t, "charlie", "charlie@ajuda.dev")

	token := validTokenFor(t, alpha.Id)
	firstPage := listUsers(t, app, "?limit=2", token)
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

	secondPage := listUsers(t, app, "?page=2&limit=2", token)
	if len(secondPage.Data) != 1 {
		t.Fatalf("esperava 1 usuário na segunda página, recebeu %d", len(secondPage.Data))
	}
	if secondPage.HasNext {
		t.Error("esperava has_next false na segunda página")
	}
	if secondPage.Data[0].Id != charlie.Id {
		t.Errorf("esperava [charlie] na segunda página, recebeu %+v", userSearchIds(secondPage))
	}
}

func TestListUsersResponseDoesNotExposeEmail(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	java := createSkillForTest(t, "JAVA")
	assignSkillViaApi(t, app, java.Id, ana.Id, skilldomain.LevelTeach)

	token := validTokenFor(t, ana.Id)
	for _, query := range []string{"?skill=JAVA", ""} {
		status, body := listUsersRaw(t, app, query, token)
		if status != fiber.StatusOK {
			t.Fatalf("query %q: esperava 200, recebeu %d (body: %s)", query, status, body)
		}
		if strings.Contains(body, "email") {
			t.Errorf("query %q: o body não deve conter 'email', recebeu %s", query, body)
		}
		if !strings.Contains(body, `"id"`) || !strings.Contains(body, `"name"`) || !strings.Contains(body, `"skills"`) {
			t.Errorf("query %q: esperava id, name e skills no item, recebeu %s", query, body)
		}
	}
}

func TestListUsersEmailFilterWorksWhileEmailHidden(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	admin := createUserWithRole(t, "list_email_hidden_admin@ajuda.dev", userdomain.UserRoleAdmin)

	status, body := listUsersRaw(t, app, "?email=ana@ajuda.dev", validTokenFor(t, admin.Id))
	if status != fiber.StatusOK {
		t.Fatalf("esperava 200 no filtro por email, recebeu %d (body: %s)", status, body)
	}
	if strings.Contains(body, "email") {
		t.Errorf("o body não deve conter 'email' mesmo filtrando por ele, recebeu %s", body)
	}
	var page userdto.PageableUserDto
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava a ana no filtro por email, recebeu %+v", page.Data)
	}
}

func TestListUsersNoMatchReturnsEmpty(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")

	page := listUsers(t, app, "?name=zzz", validTokenFor(t, ana.Id))
	if len(page.Data) != 0 {
		t.Errorf("esperava data vazio, recebeu %+v", page.Data)
	}
	if page.HasNext {
		t.Error("esperava has_next false, recebeu true")
	}
}

func TestListUsersWithoutToken(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	resp, err := app.Test(httptest.NewRequest("GET", "/v1/user", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
}

func assertEmailFilterForbidden(t *testing.T, status int, body string) {
	t.Helper()
	if status != fiber.StatusForbidden {
		t.Fatalf("esperava 403 no filtro por email, recebeu %d (body: %s)", status, body)
	}
	var respBody rest_err.RestErr
	if err := json.Unmarshal([]byte(body), &respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "only admins can filter users by email" {
		t.Errorf("esperava message 'only admins can filter users by email', recebeu '%s'", respBody.Message)
	}
	if strings.Contains(body, `"data"`) {
		t.Errorf("o 403 não deve devolver lista, recebeu %s", body)
	}
}

func TestListUsersEmailFilterForbiddenForUser(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	ana := createUserForSkillSearch(t, "ana", "ana@ajuda.dev")

	status, body := listUsersRaw(t, app, "?email=ana@ajuda.dev", validTokenFor(t, ana.Id))
	assertEmailFilterForbidden(t, status, body)
}

func TestListUsersEmailFilterForbiddenForModerator(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	moderator := createUserWithRole(t, "list_email_mod@ajuda.dev", userdomain.UserRoleModerator)

	status, body := listUsersRaw(t, app, "?email=ana@ajuda.dev", validTokenFor(t, moderator.Id))
	assertEmailFilterForbidden(t, status, body)
}

func TestListUsersEmailFilterUnknownReturnsEmptyForAdmin(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	admin := createUserWithRole(t, "list_email_unknown_admin@ajuda.dev", userdomain.UserRoleAdmin)

	page := listUsers(t, app, "?email=nao.existe@ajuda.dev", validTokenFor(t, admin.Id))
	if len(page.Data) != 0 {
		t.Errorf("esperava data vazio para email inexistente, recebeu %+v", page.Data)
	}
}

func TestListUsersEmailFilterGhostToken(t *testing.T) {
	t.Cleanup(cleanUsersTable)

	app := setupApp()
	createUserForSkillSearch(t, "ana", "ana@ajuda.dev")

	status, body := listUsersRaw(t, app, "?email=ana@ajuda.dev", validTokenFor(t, uuidv7.New().String()))
	if status != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 com token fantasma, recebeu %d (body: %s)", status, body)
	}
	var respBody rest_err.RestErr
	if err := json.Unmarshal([]byte(body), &respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}
