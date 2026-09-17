package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
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
	CreatorRole string    `json:"creator_role"`
}

func newEventRegisterRequest(body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/v1/event/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createEventOwner(t *testing.T) *userdomain.UserDomain {
	t.Helper()
	user, err := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "dono do evento",
		Email:    testEmail,
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createEventAddress(t *testing.T, city string) *addressdomain.AddressDomain {
	t.Helper()
	address, err := addressRepository.CreateAddress(&addressdomain.AddressDomain{
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

func createEventCommunity(t *testing.T, name string, owner *userdomain.UserDomain, address *addressdomain.AddressDomain) *communitydomain.CommunityDomain {
	t.Helper()
	community, err := communityRepository.CreateCommunity(&communitydomain.CommunityDomain{
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

func registerEventViaApi(t *testing.T, app *fiber.App, body eventTestRequest) eventdto.RegisterEventDto {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, body.OwnerId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201, recebeu %d (body: %+v)", resp.StatusCode, respBody)
	}
	var respDto eventdto.RegisterEventDto
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
	resp, err := doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, body.OwnerId))
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
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
		Type:        eventdomain.TypeOnline,
		Title:       "Categoria inválida",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	}, "category")

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    eventdomain.CategoryCommunityEvent,
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
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
	token := validTokenFor(t, user.Id)
	address := createEventAddress(t, "sao paulo")
	communityFuture := createEventCommunity(t, "Comunidade com evento futuro", user, address)
	communityPast := createEventCommunity(t, "Comunidade com evento passado", user, address)

	futureEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		CommunityId: strPtr(communityFuture.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento futuro da comunidade",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	pastEvent, createErr := eventRepository.CreateEvent(&eventdomain.EventDomain{
		Owner:       *user,
		Community:   &communitydomain.CommunityDomain{Id: communityPast.Id},
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento passado da comunidade",
		Description: "evento online",
		StartAt:     time.Now().Add(-24 * time.Hour),
		DurationMin: 60,
	})
	if createErr != nil {
		t.Fatalf("failed to create past event: %v", createErr)
	}

	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/event?community_id="+communityFuture.Id, nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page eventdto.PageableEventDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != futureEvent.Id {
		t.Errorf("esperava somente o evento futuro da comunidade, recebeu %+v", page.Data)
	}
	if page.HasNext {
		t.Errorf("esperava has_next false, recebeu true")
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("GET", "/v1/event?upcoming=true", nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	page = eventdto.PageableEventDto{}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Id != futureEvent.Id {
		t.Errorf("esperava somente o evento futuro na listagem upcoming, recebeu %+v", page.Data)
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("GET", "/v1/event?community_id="+communityPast.Id, nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	page = eventdto.PageableEventDto{}
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
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
		Title:       "Presencial em sao paulo",
		Description: "evento presencial",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		AddressId:   strPtr(addressCampinas.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
		Title:       "Presencial em campinas",
		Description: "evento presencial",
		StartAt:     time.Now().Add(72 * time.Hour),
		DurationMin: 60,
	})
	registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Online sem cidade",
		Description: "evento online",
		StartAt:     time.Now().Add(96 * time.Hour),
		DurationMin: 60,
	})

	resp, err := doAuthedRequest(app, httptest.NewRequest("GET", "/v1/event?city=sao%20paulo", nil), validTokenFor(t, user.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", resp.StatusCode)
	}
	var page eventdto.PageableEventDto
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
	token := validTokenFor(t, user.Id)
	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     user.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento para deletar",
		Description: "evento online",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})

	req := httptest.NewRequest("DELETE", "/v1/event/"+created.Id, bytes.NewBufferString(`{"comment":"Agenda do mentor mudou"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doAuthedRequest(app, req, token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204, recebeu %d", resp.StatusCode)
	}

	resp, err = doAuthedRequest(app, httptest.NewRequest("GET", "/v1/event/"+created.Id, nil), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no GET após delete, recebeu %d", resp.StatusCode)
	}

	var deletedEvent evententity.EventEntity
	if err := db.Unscoped().Where("id = ?", created.Id).First(&deletedEvent).Error; err != nil {
		t.Fatalf("esperava achar o evento arquivado via Unscoped, recebeu %v", err)
	}
	if deletedEvent.Id != created.Id || !deletedEvent.DeletedAt.Valid {
		t.Errorf("esperava evento com deleted_at preenchido, recebeu %+v", deletedEvent)
	}
	if deletedEvent.Comment == nil || *deletedEvent.Comment != "Agenda do mentor mudou" {
		t.Errorf("esperava comment no cancelamento, recebeu %+v", deletedEvent.Comment)
	}
}

func eventRescheduleBody(t *testing.T, startAt time.Time, comment string) string {
	t.Helper()
	payload, err := json.Marshal(eventdto.RescheduleEventDto{StartAt: startAt, Comment: comment})
	if err != nil {
		t.Fatalf("erro ao montar body de reagendamento: %v", err)
	}
	return string(payload)
}

func TestRescheduleAndCancelEventComment(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "reschedule_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "reschedule_third@ajuda.dev", userdomain.UserRoleUser)
	communityOwner := createUserWithRole(t, "reschedule_community_owner@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "reschedule_moderator@ajuda.dev", userdomain.UserRoleModerator)
	address := createEventAddress(t, "reschedule_city")
	community := createEventCommunity(t, "Comunidade reagendar", communityOwner, address)
	eaAddCommunityMember(t, community.Id, owner.Id)

	ownerToken := validTokenFor(t, owner.Id)
	firstStart := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	secondStart := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento para reagendar",
		Description: "reagendar com comment",
		StartAt:     firstStart,
		DurationMin: 60,
	})
	if event.Status != eventdomain.EventStatusPending {
		t.Fatalf("esperava PENDING no evento do membro, recebeu %s", event.Status)
	}

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, secondStart, "Sexta 15h encaixa melhor"), ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no reagendamento, recebeu %d", resp.StatusCode)
	}
	rescheduled := eaDecodeEventDto(t, resp)
	if rescheduled.Comment != "Sexta 15h encaixa melhor" {
		t.Errorf("esperava o comment do reagendamento, recebeu '%s'", rescheduled.Comment)
	}
	if !rescheduled.StartAt.UTC().Truncate(time.Second).Equal(secondStart) {
		t.Errorf("esperava start_at %s, recebeu %s", secondStart, rescheduled.StartAt)
	}
	if rescheduled.Status != eventdomain.EventStatusPending {
		t.Errorf("não esperava mudar a aprovação no reagendamento, recebeu %s", rescheduled.Status)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", ownerToken)
	got := eaDecodeEventDto(t, resp)
	if got.Comment != "Sexta 15h encaixa melhor" {
		t.Errorf("esperava o comment no GET, recebeu '%s'", got.Comment)
	}

	thirdStart := time.Now().Add(96 * time.Hour).UTC().Truncate(time.Second)
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, thirdStart, "Na verdade sábado"), ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no segundo reagendamento, recebeu %d", resp.StatusCode)
	}
	overwritten := eaDecodeEventDto(t, resp)
	if overwritten.Comment != "Na verdade sábado" {
		t.Errorf("esperava o comment sobrescrito, recebeu '%s'", overwritten.Comment)
	}
	if !overwritten.StartAt.UTC().Truncate(time.Second).Equal(thirdStart) {
		t.Errorf("esperava start_at %s no overwrite, recebeu %s", thirdStart, overwritten.StartAt)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(120*time.Hour), ""), ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 sem comment, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("comment", respBody.Causes)) == 0 {
		t.Errorf("esperava cause em comment, recebeu %+v", respBody.Causes)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(120*time.Hour), "   "), ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 com comment só de espaços, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(120*time.Hour), strings.Repeat("a", 501)), ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no comment > 500, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes := getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment must have at most 500 characters" {
		t.Errorf("esperava cause de tamanho, recebeu %+v", respBody.Causes)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(-24*time.Hour), "data passada"), ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no start_at passado, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	if len(getCauseByField("start_at", respBody.Causes)) == 0 {
		t.Errorf("esperava cause em start_at, recebeu %+v", respBody.Causes)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(120*time.Hour), "Sexta 15h encaixa melhor"), validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	communityEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento do dono da comunidade",
		Description: "reagendar pelo dono da comunidade",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+communityEvent.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(80*time.Hour), "Dono da comunidade reagendou"), validTokenFor(t, communityOwner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o dono da comunidade, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	moderatorEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento do moderador",
		Description: "reagendar pelo moderador",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+moderatorEvent.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(80*time.Hour), "Moderador reagendou"), validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o moderador, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	cancelEvent := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento para cancelar sem comment",
		Description: "deve permanecer",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+cancelEvent.Id, `{"comment":""}`, ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no cancel sem comment, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes = getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment is required when cancelling" {
		t.Errorf("esperava cause comment is required when cancelling, recebeu %+v", respBody.Causes)
	}
	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+cancelEvent.Id, "", ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava o evento permanecer após cancel 400, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(110*time.Hour), "Antes do cancel"), ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no reagendamento antes do cancel, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+event.Id, `{"comment":"Agenda do mentor mudou"}`, ownerToken)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 no cancel após reagendar, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	var deletedEvent evententity.EventEntity
	if err := db.Unscoped().Where("id = ?", event.Id).First(&deletedEvent).Error; err != nil {
		t.Fatalf("esperava achar o evento arquivado via Unscoped, recebeu %v", err)
	}
	if !deletedEvent.DeletedAt.Valid {
		t.Errorf("esperava deleted_at preenchido após o cancel")
	}
	if deletedEvent.Comment == nil || *deletedEvent.Comment != "Agenda do mentor mudou" {
		t.Errorf("esperava o comment do cancel na linha deletada, recebeu %+v", deletedEvent.Comment)
	}
}

func TestMentoringRescheduleSwapsParticipantStatus(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "mentoring_reschedule_owner@ajuda.dev", userdomain.UserRoleUser)
	mentee := createUserWithRole(t, "mentoring_reschedule_mentee@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "mentoring_reschedule_third@ajuda.dev", userdomain.UserRoleUser)
	ownerToken := validTokenFor(t, owner.Id)
	menteeToken := validTokenFor(t, mentee.Id)

	event := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, event.Id, mentee.Id, owner.Id)

	guestStart := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)
	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, guestStart, "Sexta 15h encaixa melhor"), menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 quando o convidado reagenda, recebeu %d", resp.StatusCode)
	}
	rescheduled := eaDecodeEventDto(t, resp)
	if !rescheduled.StartAt.UTC().Truncate(time.Second).Equal(guestStart) {
		t.Errorf("esperava start_at %s, recebeu %s", guestStart, rescheduled.StartAt)
	}
	assertEventUserStatus(t, event.Id, owner.Id, eventdomain.StatusRequested)
	assertEventUserStatus(t, event.Id, mentee.Id, eventdomain.StatusConfirmed)

	ownerStart := time.Now().Add(80 * time.Hour).UTC().Truncate(time.Second)
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, ownerStart, "Volto para o horário original"), ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 quando o criador reagenda, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	assertEventUserStatus(t, event.Id, owner.Id, eventdomain.StatusConfirmed)
	assertEventUserStatus(t, event.Id, mentee.Id, eventdomain.StatusRequested)

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusConfirmed, "Nos falamos pelo LinkedIn", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite do mentee, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	bothConfirmedStart := time.Now().Add(96 * time.Hour).UTC().Truncate(time.Second)
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, bothConfirmedStart, "Preciso de outro horário"), menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 quando o mentee reagenda após o aceite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	assertEventUserStatus(t, event.Id, mentee.Id, eventdomain.StatusConfirmed)
	ownerRow := euFindRow(t, event.Id, owner.Id)
	if ownerRow.Status != eventdomain.StatusRequested {
		t.Errorf("esperava o criador REQUESTED, recebeu %s", ownerRow.Status)
	}
	if ownerRow.StatusComment != nil {
		t.Errorf("esperava status_comment limpo no criador, recebeu %+v", ownerRow.StatusComment)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(110*time.Hour), "Terceiro não pode"), validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the event owner, the invited participant, the community owner or moderators can reschedule this event" {
		t.Errorf("esperava a mensagem de 403 do reschedule, recebeu '%s'", respBody.Message)
	}

	cancelledEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, cancelledEvent.Id, mentee.Id, owner.Id)
	resp = euCancelParticipation(t, app, cancelledEvent.Id, mentee.Id, menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no cancelamento, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+cancelledEvent.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(120*time.Hour), "Depois do cancel"), menteeToken)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para participante CANCELLED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCommunityEventRescheduleLeavesParticipantsUnchanged(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "community_reschedule_owner@ajuda.dev", userdomain.UserRoleUser)
	attendee := createUserWithRole(t, "community_reschedule_attendee@ajuda.dev", userdomain.UserRoleUser)
	event := euCreateCommunityEvent(t, app, owner, nil)

	join := euJoinEvent(t, app, event.Id, "", validTokenFor(t, attendee.Id))
	if join.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join, recebeu %d", join.StatusCode)
	}
	join.Body.Close()

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(80*time.Hour), "Novo horário"), validTokenFor(t, attendee.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para attendee reagendar community event, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, time.Now().Add(80*time.Hour), "Novo horário"), validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no reagendamento do dono, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	assertEventUserStatus(t, event.Id, attendee.Id, eventdomain.StatusConfirmed)
}

func assertEventUserStatus(t *testing.T, eventId string, userId string, want string) {
	t.Helper()
	row := euFindRow(t, eventId, userId)
	if row.Status != want {
		t.Errorf("esperava status %s para %s, recebeu %s", want, userId, row.Status)
	}
}
