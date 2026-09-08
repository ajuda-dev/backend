package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type eventTestRequest struct {
	OwnerId     string    `json:"owner_id"`
	CommunityId *string   `json:"community_id"`
	AddressId   *string   `json:"address_id"`
	Category    string    `json:"category"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"start_at"`
	DurationMin int       `json:"duration_min"`
	MeetingLink string    `json:"meeting_link"`
	MaxSlots    *int      `json:"max_slots"`
}

func newEventRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/event/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createEventOwner(t *testing.T) *domain.UserDomain {
	t.Helper()
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name:     "dono do evento",
		Email:    testEmail,
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createEventAddress(t *testing.T, city string) *domain.AddressDomain {
	t.Helper()
	address, err := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    city,
		State:   "sp",
		Street:  "rua teste",
		ZipCode: "01000-000",
	})
	if err != nil {
		t.Fatalf("failed to create address: %v", err)
	}
	return address
}

func createEventCommunity(t *testing.T, name string, owner *domain.UserDomain, address *domain.AddressDomain) *domain.CommunityDomain {
	t.Helper()
	community, err := communityRepository.CreateCommunity(&domain.CommunityDomain{
		Name:        name,
		Description: "comunidade de teste de eventos",
		Owner:       *owner,
		Address:     *address,
	})
	if err != nil {
		t.Fatalf("failed to create community: %v", err)
	}
	return community
}

func strPtr(s string) *string {
	return &s
}

func registerEventViaApi(t *testing.T, app *fiber.App, body eventTestRequest) dto.RegisterEventDto {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := app.Test(newEventRegisterRequest(payload))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201, recebeu %d (body: %+v)", resp.StatusCode, respBody)
	}
	var respDto dto.RegisterEventDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func eventReqValidation(t *testing.T, app *fiber.App, body eventTestRequest, expectedCauseField string) rest_err.RestErr {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := app.Test(newEventRegisterRequest(payload))
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
	causes := getCauseByField(expectedCauseField, respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava cause para o campo '%s', recebeu %+v", expectedCauseField, respBody.Causes)
	}
	return respBody
}

func TestCreateOnlineEventSuccess(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	respDto := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Encontro online da comunidade",
		Description: "evento online de testes",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})

	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
	if respDto.OwnerId != user.Id {
		t.Errorf("esperava owner_id '%s', recebeu '%s'", user.Id, respDto.OwnerId)
	}
	if respDto.AddressId != nil {
		t.Errorf("esperava address_id nulo para evento ONLINE, recebeu %v", *respDto.AddressId)
	}
}

func TestCreateInpersonEventRequiresAddress(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeInperson,
		Title:       "Encontro presencial",
		Description: "evento presencial sem endereço",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "address_id")
}

func TestCreateInpersonEventWithInexistentAddress(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	respBody := eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		AddressId:   strPtr(uuidv7.New().String()),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeInperson,
		Title:       "Encontro presencial",
		Description: "evento presencial com endereço inexistente",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "address_id")

	causes := getCauseByField("address_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "address_id is not valid, not found this address" {
		t.Errorf("esperava cause de endereço não encontrado, recebeu %+v", respBody.Causes)
	}
}

func TestCreateEventWithInexistentCommunity(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	respBody := eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		CommunityId: strPtr(uuidv7.New().String()),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento de comunidade inexistente",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "community_id")

	causes := getCauseByField("community_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "community_id is not valid, not found this community" {
		t.Errorf("esperava cause de comunidade não encontrada, recebeu %+v", respBody.Causes)
	}
}

func TestCreateEventInvalidCategoryAndType(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    "MEETUP",
		Type:        domain.TypeOnline,
		Title:       "Categoria inválida",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "category")

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        "PHYSICAL",
		Title:       "Tipo inválido",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "type")
}

func TestCreateEventWithPastStart(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento no passado",
		Description: "evento online",
		StartAt:     time.Now().Add(-24 * time.Hour),
		DurationMin: 60,
	}, "start_at")
}

func TestListEventsByCommunityAndUpcoming(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)
	address := createEventAddress(t, "sao paulo")
	communityFuture := createEventCommunity(t, "Comunidade com evento futuro", user, address)
	communityPast := createEventCommunity(t, "Comunidade com evento passado", user, address)

	futureEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		CommunityId: strPtr(communityFuture.Id),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento futuro da comunidade",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	pastEvent, createErr := eventRepository.CreateEvent(&domain.EventDomain{
		Owner:       *user,
		Community:   &domain.CommunityDomain{Id: communityPast.Id},
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento passado da comunidade",
		Description: "evento online",
		StartAt:     time.Now().Add(-24 * time.Hour),
		DurationMin: 60,
	})
	if createErr != nil {
		t.Fatalf("failed to create past event: %v", createErr)
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/event?community_id="+communityFuture.Id, nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page dto.PageableEventDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != futureEvent.Id {
		t.Errorf("esperava somente o evento futuro da comunidade, recebeu %+v", page.Data)
	}
	if page.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/v1/event?upcoming=true", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	page = dto.PageableEventDto{}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != futureEvent.Id {
		t.Errorf("esperava somente o evento futuro na listagem upcoming, recebeu %+v", page.Data)
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/v1/event?community_id="+communityPast.Id, nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	page = dto.PageableEventDto{}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != pastEvent.Id {
		t.Errorf("esperava somente o evento da outra comunidade, recebeu %+v", page.Data)
	}
}

func TestListEventsByCity(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)
	addressSaoPaulo := createEventAddress(t, "sao paulo")
	addressCampinas := createEventAddress(t, "campinas")

	spEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		AddressId:   strPtr(addressSaoPaulo.Id),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeInperson,
		Title:       "Presencial em sao paulo",
		Description: "evento presencial",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		AddressId:   strPtr(addressCampinas.Id),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeInperson,
		Title:       "Presencial em campinas",
		Description: "evento presencial",
		StartAt:     time.Now().Add(72 * time.Hour),
		DurationMin: 60,
	})
	registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Online sem cidade",
		Description: "evento online",
		StartAt:     time.Now().Add(96 * time.Hour),
		DurationMin: 60,
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/event?city=sao%20paulo", nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page dto.PageableEventDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != spEvent.Id {
		t.Errorf("esperava somente o evento presencial de sao paulo, recebeu %+v", page.Data)
	}
}

func TestDeleteEventSoftDelete(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)
	t.Cleanup(cleanEventsTable)

	app := setupApp()
	user := createEventOwner(t)
	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento para deletar",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})

	req := httptest.NewRequest("DELETE", "/v1/event/"+created.Id, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/v1/event/"+created.Id, nil))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no GET após delete, recebeu %d", resp.StatusCode)
	}

	var deletedEvent entity.EventEntity
	if err := db.Unscoped().Where("id = ?", created.Id).First(&deletedEvent).Error; err != nil {
		t.Fatalf("esperava achar o evento arquivado via Unscoped, recebeu %v", err)
	}
	if deletedEvent.Id != created.Id || !deletedEvent.DeletedAt.Valid {
		t.Errorf("esperava evento com deleted_at preenchido, recebeu %+v", deletedEvent)
	}
}
