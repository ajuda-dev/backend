package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func eaDecodeEventDto(t *testing.T, resp *http.Response) eventdto.EventDto {
	t.Helper()
	defer resp.Body.Close()
	var eventDto eventdto.EventDto
	if err := json.NewDecoder(resp.Body).Decode(&eventDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return eventDto
}

func eaDecodePageableEvent(t *testing.T, resp *http.Response) eventdto.PageableEventDto {
	t.Helper()
	defer resp.Body.Close()
	var page eventdto.PageableEventDto
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return page
}

func TestDeleteEventAuthorization(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "del_community_owner@ajuda.dev", userdomain.UserRoleUser)
	eventOwner := createUserWithRole(t, "del_event_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "del_third@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "del_moderator@ajuda.dev", userdomain.UserRoleModerator)
	address := createEventAddress(t, "del_city")
	community := createEventCommunity(t, "Comunidade delete", communityOwner, address)
	eaAddCommunityMember(t, community.Id, eventOwner.Id)

	createEvent := func(title string) eventdto.RegisterEventDto {
		t.Helper()
		return registerEventViaApi(t, app, eventTestRequest{
			OwnerId:     eventOwner.Id,
			CommunityId: strPtr(community.Id),
			Category:    eventdomain.CategoryCommunityEvent,
			Type:        eventdomain.TypeOnline,
			Title:       title,
			Description: "teste de delete",
			StartAt:     time.Now().Add(48 * time.Hour),
			DurationMin: 60,
		})
	}

	resp := euRequest(t, app, http.MethodDelete, "/v1/event/invalid-id", "", validTokenFor(t, eventOwner.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 para uuid inválido, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+uuidv7.New().String(), "", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	cancelBody := `{"comment":"Agenda do mentor mudou"}`
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+uuidv7.New().String(), cancelBody, validTokenFor(t, eventOwner.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 para evento inexistente, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	eventForThird := createEvent("Evento negado ao terceiro")
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+eventForThird.Id, cancelBody, validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the event owner, the community owner or moderators can manage this event" {
		t.Errorf("esperava a mensagem de 403 do canManageEvent, recebeu '%s'", respBody.Message)
	}

	eventForCommunityOwner := createEvent("Evento do dono da comunidade")
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+eventForCommunityOwner.Id, cancelBody, validTokenFor(t, communityOwner.Id))
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 para o dono da comunidade, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	eventForModerator := createEvent("Evento do moderador")
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+eventForModerator.Id, cancelBody, validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 para o moderador, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	eventForOwner := createEvent("Evento do criador")
	ownerToken := validTokenFor(t, eventOwner.Id)
	resp = euRequest(t, app, http.MethodDelete, "/v1/event/"+eventForOwner.Id, cancelBody, ownerToken)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 para o criador, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+eventForOwner.Id, "", ownerToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no GET após o soft delete, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	var deletedEvent evententity.EventEntity
	if err := db.Unscoped().Where("id = ?", eventForOwner.Id).First(&deletedEvent).Error; err != nil {
		t.Fatalf("esperava achar o evento arquivado via Unscoped, recebeu %v", err)
	}
	if deletedEvent.Id != eventForOwner.Id || !deletedEvent.DeletedAt.Valid {
		t.Errorf("esperava evento com deleted_at preenchido, recebeu %+v", deletedEvent)
	}
}

func TestEventAgendaOtherUser(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "agenda_owner@ajuda.dev", userdomain.UserRoleUser)
	other := createUserWithRole(t, "agenda_other@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "agenda_moderator@ajuda.dev", userdomain.UserRoleModerator)
	event := euCreateCommunityEvent(t, app, owner, nil)

	resp := euRequest(t, app, http.MethodGet, "/v1/event?user_id="+owner.Id, "", validTokenFor(t, other.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403 na agenda alheia, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only moderators and admins can read another user's agenda" {
		t.Errorf("esperava a mensagem de 403 da agenda alheia, recebeu '%s'", respBody.Message)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event?user_id="+owner.Id, "", validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 na própria agenda, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event?user_id="+owner.Id, "", validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 na agenda alheia para moderador, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", validTokenFor(t, other.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 na listagem sem filtro, recebeu %d", resp.StatusCode)
	}
	page := eaDecodePageableEvent(t, resp)
	if len(page.Data) != 1 || page.Data[0].Id != event.Id {
		t.Errorf("esperava o evento criado na listagem, recebeu %+v", page.Data)
	}
}

func TestEventOwnerEmailFiltered(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "event_email_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "event_email_third@ajuda.dev", userdomain.UserRoleUser)
	event := euCreateCommunityEvent(t, app, owner, nil)
	thirdToken := validTokenFor(t, third.Id)

	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe do evento, recebeu %d", resp.StatusCode)
	}
	eventDto := eaDecodeEventDto(t, resp)
	if eventDto.Owner == nil || eventDto.Owner.Email != "" {
		t.Errorf("esperava o email do owner filtrado para o terceiro, recebeu %+v", eventDto.Owner)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	page := eaDecodePageableEvent(t, resp)
	if len(page.Data) != 1 || page.Data[0].Owner == nil || page.Data[0].Owner.Email != "" {
		t.Errorf("esperava o email do owner filtrado na listagem, recebeu %+v", page.Data)
	}

	euShareEmail(t, app, owner)

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe do evento, recebeu %d", resp.StatusCode)
	}
	eventDto = eaDecodeEventDto(t, resp)
	if eventDto.Owner == nil || eventDto.Owner.Email != owner.Email {
		t.Errorf("esperava o email compartilhado visível ao terceiro, recebeu %+v", eventDto.Owner)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe do evento para o owner, recebeu %d", resp.StatusCode)
	}
	eventDto = eaDecodeEventDto(t, resp)
	if eventDto.Owner == nil || eventDto.Owner.Email != owner.Email {
		t.Errorf("esperava o próprio email completo para o owner, recebeu %+v", eventDto.Owner)
	}
}
