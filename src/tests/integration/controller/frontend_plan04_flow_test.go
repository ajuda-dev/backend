package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Verificação end-to-end do plano 04 do frontend: o fluxo que as telas de
// explorar/detalhe/membership executam contra a API real.
func TestFrontendPlan04Flow(t *testing.T) {
	t.Cleanup(cleanCommunityUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)

	app := setupApp()

	owner := createCommunityUser(t)
	ownerToken := validTokenFor(t, owner.Id)

	address := createCommunityAddress(t, "sao paulo")
	communityId := registerCommunityViaApi(t, app, ownerToken, address.Id, "Comunidade Plano 04")

	// 1) Explorar: GET /v1/community?page=1&limit=10
	req := httptest.NewRequest("GET", "/v1/community?page=1&limit=10", nil)
	resp, err := doAuthedRequest(app, req, ownerToken)
	if err != nil {
		t.Fatalf("erro ao listar comunidades: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado 200 na listagem, obtido %d", resp.StatusCode)
	}
	var page dto.PageableCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar listagem: %v", err)
	}
	if len(page.Data) == 0 {
		t.Fatal("listagem vazia: o card de explorar não teria o que renderizar")
	}
	found := false
	for _, item := range page.Data {
		if item.Id == communityId {
			found = true
			if item.Name != "Comunidade Plano 04" {
				t.Fatalf("nome inesperado no card: %q", item.Name)
			}
			if item.Address.City != "sao paulo" {
				t.Fatalf("cidade inesperada no card: %q", item.Address.City)
			}
			if item.Owner.Id != owner.Id {
				t.Fatalf("owner inesperado no card: %q", item.Owner.Id)
			}
		}
	}
	if !found {
		t.Fatalf("comunidade %s não apareceu na listagem", communityId)
	}

	// 2) Filtro por cidade em minúsculas (como o service do frontend envia)
	req = httptest.NewRequest("GET", "/v1/community?page=1&limit=10&city=sao%20paulo", nil)
	resp, err = doAuthedRequest(app, req, ownerToken)
	if err != nil {
		t.Fatalf("erro ao filtrar por cidade: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado 200 no filtro, obtido %d", resp.StatusCode)
	}
	var filtered dto.PageableCommunityDto
	if err := json.NewDecoder(resp.Body).Decode(&filtered); err != nil {
		t.Fatalf("erro ao decodificar filtro: %v", err)
	}
	if len(filtered.Data) == 0 {
		t.Fatal("filtro por cidade não retornou a comunidade criada")
	}

	// 3) Detalhe: o frontend busca a comunidade percorrendo a listagem
	detail := findCommunityInPages(t, app, ownerToken, communityId)
	if detail.Name != "Comunidade Plano 04" {
		t.Fatalf("detalhe com nome inesperado: %q", detail.Name)
	}
	if detail.Description == "" {
		t.Fatal("detalhe sem descrição")
	}

	// 4) Membership: outro usuário entra e sai
	member := createUserWithRole(t, "membro.plano04@ajudadev.dev", domain.UserRoleUser)
	memberToken := validTokenFor(t, member.Id)

	joinReq := httptest.NewRequest("POST", "/v1/community/"+communityId+"/join", nil)
	joinResp, err := doAuthedRequest(app, joinReq, memberToken)
	if err != nil {
		t.Fatalf("erro ao entrar na comunidade: %v", err)
	}
	if joinResp.StatusCode != http.StatusCreated {
		t.Fatalf("esperado 201 no join, obtido %d", joinResp.StatusCode)
	}
	var membership dto.CommunityUserDto
	if err := json.NewDecoder(joinResp.Body).Decode(&membership); err != nil {
		t.Fatalf("erro ao decodificar membership: %v", err)
	}
	if membership.CommunityId != communityId || membership.UserId != member.Id {
		t.Fatalf("membership inesperado: %+v", membership)
	}

	// Join repetido: o frontend trata 400 como "já é membro"
	dupReq := httptest.NewRequest("POST", "/v1/community/"+communityId+"/join", nil)
	dupResp, err := doAuthedRequest(app, dupReq, memberToken)
	if err != nil {
		t.Fatalf("erro no join duplicado: %v", err)
	}
	if dupResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperado 400 no join duplicado, obtido %d", dupResp.StatusCode)
	}

	leaveReq := httptest.NewRequest("DELETE", "/v1/community/"+communityId+"/leave", nil)
	leaveResp, err := doAuthedRequest(app, leaveReq, memberToken)
	if err != nil {
		t.Fatalf("erro ao sair da comunidade: %v", err)
	}
	if leaveResp.StatusCode != http.StatusNoContent {
		t.Fatalf("esperado 204 no leave, obtido %d", leaveResp.StatusCode)
	}

	// Leave repetido: o frontend trata 404 como "não era membro"
	leaveAgainReq := httptest.NewRequest("DELETE", "/v1/community/"+communityId+"/leave", nil)
	leaveAgainResp, err := doAuthedRequest(app, leaveAgainReq, memberToken)
	if err != nil {
		t.Fatalf("erro no leave duplicado: %v", err)
	}
	if leaveAgainResp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperado 404 no leave duplicado, obtido %d", leaveAgainResp.StatusCode)
	}

	// 5) Detalhe de id inexistente: o frontend mostra "Comunidade não encontrada."
	missing := findCommunityInPages(t, app, ownerToken, uuid.NewString())
	if missing != nil {
		t.Fatal("id inexistente deveria resultar em comunidade não encontrada")
	}
}

func findCommunityInPages(t *testing.T, app *fiber.App, token string, id string) *dto.CommunityDto {
	t.Helper()
	for page := 1; page <= 20; page++ {
		req := httptest.NewRequest("GET", "/v1/community?page="+strconv.Itoa(page)+"&limit=100", nil)
		resp, err := doAuthedRequest(app, req, token)
		if err != nil {
			t.Fatalf("erro ao buscar página %d: %v", page, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("esperado 200 na página %d, obtido %d", page, resp.StatusCode)
		}
		var body dto.PageableCommunityDto
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("erro ao decodificar página %d: %v", page, err)
		}
		for i := range body.Data {
			if body.Data[i].Id == id {
				return &body.Data[i]
			}
		}
		if !body.HasNext {
			return nil
		}
	}
	return nil
}
