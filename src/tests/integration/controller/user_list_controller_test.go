package controller_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func listUsersRaw(t *testing.T, app *fiber.App, query string) (int, string) {
	t.Helper()
	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/user"+query, nil), validTokenFor(t, uuidv7.New().String()))
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

func listUsers(t *testing.T, app *fiber.App, query string) dto.PageableUserDto {
	t.Helper()
	status, body := listUsersRaw(t, app, query)
	if status != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem de usuários, recebeu %d (body: %s)", status, body)
	}
	var page dto.PageableUserDto
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func userNames(page dto.PageableUserDto) []string {
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
	assignSkillViaApi(t, app, java.Id, ana.Id, domain.LevelTeach)
	assignSkillViaApi(t, app, goSkill.Id, bruno.Id, domain.LevelWantToLearn)

	page := listUsers(t, app, "")
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
	createUserForSkillSearch(t, "ana", "ana@ajuda.dev")
	createUserForSkillSearch(t, "bruno", "bruno@ajuda.dev")

	for _, query := range []string{"", "?skill=", "?skill=%20%20"} {
		page := listUsers(t, app, query)
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

	page := listUsers(t, app, "")
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

	page := listUsers(t, app, "?name=an")
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

	for _, query := range []string{"?name=ANA", "?name=ana%20SILVA", "?name=%20ana"} {
		page := listUsers(t, app, query)
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

	for _, query := range []string{"?name=jose", "?name=JOSÉ", "?name=josé"} {
		page := listUsers(t, app, query)
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

	page := listUsers(t, app, "?name=josé")
	if len(page.Data) != 1 || page.Data[0].Id != jose.Id {
		t.Errorf("esperava o usuário 'Jose' para termo acentuado, recebeu %+v", page.Data)
	}
}

func TestListUsersByNameEscape(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanSkillsTable)
	t.Cleanup(cleanSkillUsersTable)

	app := setupApp()
	createUserForSkillSearch(t, "ana silva", "ana.silva@ajuda.dev")

	wildcardPage := listUsers(t, app, "?name=%25")
	if len(wildcardPage.Data) != 0 {
		t.Errorf("esperava 0 resultados para '%%' literal, recebeu %+v", wildcardPage.Data)
	}

	underscorePage := listUsers(t, app, "?name=ana_silva")
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
	assignSkillViaApi(t, app, java.Id, ana.Id, domain.LevelTeach)

	page := listUsers(t, app, "?skill=JAVA&name=ana&email=ana.silva@ajuda.dev")
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava somente a ana com os 3 filtros combinados, recebeu %+v", page.Data)
	}

	noMatch := listUsers(t, app, "?skill=JAVA&name=ana&email=ana.souza@ajuda.dev")
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

	page := listUsers(t, app, "?email=ANA.SILVA@AJUDA.DEV")
	if len(page.Data) != 1 || page.Data[0].Id != ana.Id {
		t.Errorf("esperava a ana no match exato case-insensitive, recebeu %+v", page.Data)
	}

	for _, query := range []string{"?email=ana.silva", "?email=ajuda.dev", "?email=nao.existe@ajuda.dev"} {
		partial := listUsers(t, app, query)
		if len(partial.Data) != 0 {
			t.Errorf("query %q: esperava vazio (match exato), recebeu %+v", query, partial.Data)
		}
	}

	blank := listUsers(t, app, "?email=")
	if len(blank.Data) != 2 {
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

	firstPage := listUsers(t, app, "?limit=2")
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

	secondPage := listUsers(t, app, "?page=2&limit=2")
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
	assignSkillViaApi(t, app, java.Id, ana.Id, domain.LevelTeach)

	for _, query := range []string{"?skill=JAVA", ""} {
		status, body := listUsersRaw(t, app, query)
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

	status, body := listUsersRaw(t, app, "?email=ana@ajuda.dev")
	if status != fiber.StatusOK {
		t.Fatalf("esperava 200 no filtro por email, recebeu %d (body: %s)", status, body)
	}
	if strings.Contains(body, "email") {
		t.Errorf("o body não deve conter 'email' mesmo filtrando por ele, recebeu %s", body)
	}
	var page dto.PageableUserDto
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
	createUserForSkillSearch(t, "ana", "ana@ajuda.dev")

	page := listUsers(t, app, "?name=zzz")
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
