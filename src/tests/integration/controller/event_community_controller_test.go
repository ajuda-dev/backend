package controller_test

import (
	"net/http"
	"testing"
	"time"

	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

func TestCommunityEventStartsClosedAndHidden(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "closed_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "closed_third@ajuda.dev", userdomain.UserRoleUser)
	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento fechado",
		Description: "ainda não público",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if event.Visibility != eventdomain.EventVisibilityClosed {
		t.Fatalf("esperava CLOSED, recebeu %s", event.Visibility)
	}

	thirdToken := validTokenFor(t, third.Id)
	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no detalhe fechado para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euJoinEvent(t, app, event.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no join de evento fechado, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("event_id", respBody.Causes)) == 0 || getCauseByField("event_id", respBody.Causes)[0] != "event is not public yet" {
		t.Errorf("esperava cause de evento não público, recebeu %+v", respBody.Causes)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	page := eaDecodePageableEvent(t, resp)
	for _, item := range page.Data {
		if item.Id == event.Id {
			t.Errorf("não esperava o evento fechado na listagem pública, recebeu %+v", item)
		}
	}
}

func TestCommunityEventSingleSpeakerAndConflict(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "conflict_community_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "conflict_member@ajuda.dev", userdomain.UserRoleUser)
	speaker := createUserWithRole(t, "conflict_speaker@ajuda.dev", userdomain.UserRoleUser)
	otherSpeaker := createUserWithRole(t, "conflict_other_speaker@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "conflict_city")
	community := createEventCommunity(t, "Comunidade conflito", communityOwner, address)
	eaAddCommunityMember(t, community.Id, member.Id)

	start := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	first := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     communityOwner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Primeiro horário",
		Description: "ocupa a agenda",
		StartAt:     start,
		DurationMin: 60,
	})
	ownerToken := validTokenFor(t, communityOwner.Id)
	resp := euAddParticipant(t, app, first.Id, `{"user_id":"`+speaker.Id+`","role":"SPEAKER"}`, ownerToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no primeiro convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = euAddParticipant(t, app, first.Id, `{"user_id":"`+otherSpeaker.Id+`","role":"SPEAKER"}`, ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no segundo palestrante, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("role", respBody.Causes)) == 0 || getCauseByField("role", respBody.Causes)[0] != "community event allows only one speaker" {
		t.Errorf("esperava cause de um palestrante, recebeu %+v", respBody.Causes)
	}

	payload := eventTestRequest{
		OwnerId:     member.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Choque de horário",
		Description: "mesmo slot",
		StartAt:     start.Add(30 * time.Minute),
		DurationMin: 60,
	}
	respBody = eventReqValidation(t, app, payload, "start_at")
	if len(getCauseByField("start_at", respBody.Causes)) == 0 || getCauseByField("start_at", respBody.Causes)[0] != "community already has an event at that time" {
		t.Errorf("esperava cause de conflito, recebeu %+v", respBody.Causes)
	}

	second := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     member.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Horário livre",
		Description: "depois do primeiro",
		StartAt:     start.Add(2 * time.Hour),
		DurationMin: 60,
	})
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+second.Id+"/reschedule",
		eventRescheduleBody(t, start.Add(15*time.Minute), "Quero o horário ocupado"), validTokenFor(t, member.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no reagendamento em conflito, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	if len(getCauseByField("start_at", respBody.Causes)) == 0 || getCauseByField("start_at", respBody.Causes)[0] != "community already has an event at that time" {
		t.Errorf("esperava cause de conflito no reagendamento, recebeu %+v", respBody.Causes)
	}
}

func TestCommunityEventPublishFlow(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "publish_community_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "publish_member@ajuda.dev", userdomain.UserRoleUser)
	speaker := createUserWithRole(t, "publish_speaker@ajuda.dev", userdomain.UserRoleUser)
	attendee := createUserWithRole(t, "publish_attendee@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "publish_city")
	community := createEventCommunity(t, "Comunidade publicar", communityOwner, address)
	eaAddCommunityMember(t, community.Id, member.Id)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     member.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento para publicar",
		Description: "fluxo fechado",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	memberToken := validTokenFor(t, member.Id)
	speakerToken := validTokenFor(t, speaker.Id)
	communityOwnerToken := validTokenFor(t, communityOwner.Id)

	resp := euPublishEvent(t, app, event.Id, memberToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 ao publicar sem palestrante confirmado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+speaker.Id+`","role":"SPEAKER"}`, memberToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+event.Id, "", speakerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no detalhe fechado para o palestrante, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, speaker.Id, eventdomain.StatusConfirmed, "", speakerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite em evento PENDING/CLOSED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	newStart := time.Now().Add(80 * time.Hour).UTC().Truncate(time.Second)
	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+event.Id+"/reschedule",
		eventRescheduleBody(t, newStart, "Preciso de outro horário"), speakerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no reagendamento do palestrante, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	assertEventUserStatus(t, event.Id, member.Id, eventdomain.StatusRequested)

	resp = euPublishEvent(t, app, event.Id, memberToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 ao publicar sem o owner aceitar o horário, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("visibility", respBody.Causes)) == 0 || getCauseByField("visibility", respBody.Causes)[0] != "event owner must accept the new time" {
		t.Errorf("esperava cause de aceite do owner, recebeu %+v", respBody.Causes)
	}

	resp = euUpdateStatus(t, app, event.Id, member.Id, eventdomain.StatusConfirmed, "", memberToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite do horário pelo owner, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euPublishEvent(t, app, event.Id, communityOwnerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na publicação pelo dono da comunidade, recebeu %d", resp.StatusCode)
	}
	published := eaDecodeEventDto(t, resp)
	if published.Visibility != eventdomain.EventVisibilityPublic {
		t.Errorf("esperava PUBLIC, recebeu %s", published.Visibility)
	}
	if published.Status != eventdomain.EventStatusApproved {
		t.Errorf("esperava APPROVED no mesmo clique do dono da comunidade, recebeu %s", published.Status)
	}

	resp = euJoinEvent(t, app, event.Id, "", validTokenFor(t, attendee.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no join após publicar e aprovar, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}
