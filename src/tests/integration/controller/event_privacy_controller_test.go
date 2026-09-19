package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

const privacyMeetingLink = "https://meet.google.com/privacy-room"

func decodeEventRaw(t *testing.T, resp *http.Response) (eventdto.EventDto, map[string]any) {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	var eventDto eventdto.EventDto
	if err := json.Unmarshal(body, &eventDto); err != nil {
		t.Fatalf("erro ao decodificar EventDto: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("erro ao decodificar JSON cru: %v", err)
	}
	return eventDto, raw
}

func decodePageRaw(t *testing.T, resp *http.Response) (eventdto.PageableEventDto, []map[string]any) {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("erro ao ler body: %v", err)
	}
	var page eventdto.PageableEventDto
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("erro ao decodificar PageableEventDto: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("erro ao decodificar JSON cru: %v", err)
	}
	items, _ := raw["data"].([]any)
	rawItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("esperava objeto no data, recebeu %T", item)
		}
		rawItems = append(rawItems, asMap)
	}
	return page, rawItems
}

func findEventRaw(items []map[string]any, id string) map[string]any {
	for _, item := range items {
		if item["id"] == id {
			return item
		}
	}
	return nil
}

func assertMeetingLinkPresent(t *testing.T, raw map[string]any, want string) {
	t.Helper()
	value, exists := raw["meeting_link"]
	if !exists {
		t.Errorf("esperava meeting_link %q, chave ausente", want)
		return
	}
	if value != want {
		t.Errorf("esperava meeting_link %q, recebeu %v", want, value)
	}
}

func assertMeetingLinkOmitted(t *testing.T, raw map[string]any) {
	t.Helper()
	value, exists := raw["meeting_link"]
	if exists && value != "" {
		t.Errorf("não esperava meeting_link, recebeu %v", value)
	}
}

func assertEventAbsent(t *testing.T, items []map[string]any, id string) {
	t.Helper()
	if found := findEventRaw(items, id); found != nil {
		t.Errorf("não esperava o evento %s na listagem, recebeu %+v", id, found)
	}
}

func createPrivacyMentoring(t *testing.T, app *fiber.App, owner *userdomain.UserDomain) eventdto.RegisterEventDto {
	t.Helper()
	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "Mentoria privada",
		Description: "1:1 fora do catálogo",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: privacyMeetingLink,
	})
	if created.Visibility != eventdomain.EventVisibilityClosed {
		t.Fatalf("esperava visibility CLOSED no create MENTORING, recebeu %s", created.Visibility)
	}
	if created.MeetingLink != privacyMeetingLink {
		t.Fatalf("esperava meeting_link no 201 do criador, recebeu %q", created.MeetingLink)
	}
	return created
}

func createPrivacyWebinar(t *testing.T, app *fiber.App, owner *userdomain.UserDomain) eventdto.RegisterEventDto {
	t.Helper()
	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar público",
		Description: "catálogo sem URL da sala",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: privacyMeetingLink,
	})
	if created.Visibility != eventdomain.EventVisibilityPublic {
		t.Fatalf("esperava visibility PUBLIC no create WEBINAR, recebeu %s", created.Visibility)
	}
	if created.MeetingLink != privacyMeetingLink {
		t.Fatalf("esperava meeting_link no 201 do criador, recebeu %q", created.MeetingLink)
	}
	return created
}

func TestMentoringPrivateAndMeetingLinkRedacted(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "privacy_mentoring_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "privacy_mentoring_third@ajuda.dev", userdomain.UserRoleUser)
	guest := createUserWithRole(t, "privacy_mentoring_guest@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "privacy_mentoring_mod@ajuda.dev", userdomain.UserRoleModerator)
	ownerToken := validTokenFor(t, owner.Id)
	thirdToken := validTokenFor(t, third.Id)
	guestToken := validTokenFor(t, guest.Id)
	modToken := validTokenFor(t, moderator.Id)

	mentoring := createPrivacyMentoring(t, app, owner)

	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+mentoring.Id, "", ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe para o owner, recebeu %d", resp.StatusCode)
	}
	_, ownerRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, ownerRaw, privacyMeetingLink)

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+mentoring.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("esperava 404 no detalhe MENTORING para terceiro, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "event not found" {
		t.Errorf("esperava 404 event not found, recebeu '%s'", respBody.Message)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	_, catalog := decodePageRaw(t, resp)
	assertEventAbsent(t, catalog, mentoring.Id)

	resp = euGetParticipants(t, app, mentoring.Id, thirdToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 nos participantes MENTORING para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	euInviteMentee(t, app, mentoring.Id, guest.Id, owner.Id)
	resp = euUpdateStatus(t, app, mentoring.Id, guest.Id, eventdomain.StatusConfirmed, "", guestToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite do convidado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+mentoring.Id, "", guestToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe para o convidado CONFIRMED, recebeu %d", resp.StatusCode)
	}
	_, guestRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, guestRaw, privacyMeetingLink)

	resp = euRequest(t, app, http.MethodGet, "/v1/event?user_id="+guest.Id, "", guestToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na agenda do convidado, recebeu %d", resp.StatusCode)
	}
	agenda, agendaRaw := decodePageRaw(t, resp)
	foundInAgenda := false
	for _, item := range agenda.Data {
		if item.Id == mentoring.Id {
			foundInAgenda = true
		}
	}
	if !foundInAgenda {
		t.Errorf("esperava a mentoria na agenda do convidado, recebeu %+v", agenda.Data)
	}
	if raw := findEventRaw(agendaRaw, mentoring.Id); raw != nil {
		assertMeetingLinkPresent(t, raw, privacyMeetingLink)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+mentoring.Id, "", modToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe MENTORING para moderador, recebeu %d", resp.StatusCode)
	}
	_, modRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, modRaw, privacyMeetingLink)
}

func TestWebinarCatalogHidesMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "privacy_webinar_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "privacy_webinar_third@ajuda.dev", userdomain.UserRoleUser)
	attendee := createUserWithRole(t, "privacy_webinar_attendee@ajuda.dev", userdomain.UserRoleUser)
	ownerToken := validTokenFor(t, owner.Id)
	thirdToken := validTokenFor(t, third.Id)
	attendeeToken := validTokenFor(t, attendee.Id)

	webinar := createPrivacyWebinar(t, app, owner)

	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+webinar.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe WEBINAR para terceiro, recebeu %d", resp.StatusCode)
	}
	eventDto, thirdRaw := decodeEventRaw(t, resp)
	if eventDto.Title != webinar.Title {
		t.Errorf("esperava o webinar visível no catálogo, recebeu %+v", eventDto)
	}
	assertMeetingLinkOmitted(t, thirdRaw)

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	_, catalog := decodePageRaw(t, resp)
	listed := findEventRaw(catalog, webinar.Id)
	if listed == nil {
		t.Fatalf("esperava o WEBINAR na listagem pública, recebeu %+v", catalog)
	}
	assertMeetingLinkOmitted(t, listed)

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+webinar.Id, "", ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe para o owner, recebeu %d", resp.StatusCode)
	}
	_, ownerRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, ownerRaw, privacyMeetingLink)

	resp = euJoinEvent(t, app, webinar.Id, "", attendeeToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join de WEBINAR, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+webinar.Id, "", attendeeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe para o participante, recebeu %d", resp.StatusCode)
	}
	_, attendeeRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, attendeeRaw, privacyMeetingLink)

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", attendeeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem do participante, recebeu %d", resp.StatusCode)
	}
	_, attendeeCatalog := decodePageRaw(t, resp)
	listed = findEventRaw(attendeeCatalog, webinar.Id)
	if listed == nil {
		t.Fatalf("esperava o WEBINAR na listagem do participante")
	}
	assertMeetingLinkPresent(t, listed, privacyMeetingLink)
}

func TestCommunityEventPublicRedactsMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "privacy_community_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "privacy_community_third@ajuda.dev", userdomain.UserRoleUser)
	thirdToken := validTokenFor(t, third.Id)

	closed := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento ainda fechado",
		Description: "regressão CLOSED",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: privacyMeetingLink,
	})
	if closed.Visibility != eventdomain.EventVisibilityClosed {
		t.Fatalf("esperava CLOSED no COMMUNITY_EVENT, recebeu %s", closed.Visibility)
	}

	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+closed.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no COMMUNITY_EVENT CLOSED para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	_, catalog := decodePageRaw(t, resp)
	assertEventAbsent(t, catalog, closed.Id)

	published := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento publicado",
		Description: "catálogo sem link",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: privacyMeetingLink,
	})
	forceEventPublic(t, published.Id)

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+published.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no COMMUNITY_EVENT PUBLIC para terceiro, recebeu %d", resp.StatusCode)
	}
	_, publishedRaw := decodeEventRaw(t, resp)
	assertMeetingLinkOmitted(t, publishedRaw)

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d", resp.StatusCode)
	}
	_, catalog = decodePageRaw(t, resp)
	listed := findEventRaw(catalog, published.Id)
	if listed == nil {
		t.Fatalf("esperava o COMMUNITY_EVENT PUBLIC na listagem")
	}
	assertMeetingLinkOmitted(t, listed)
	assertEventAbsent(t, catalog, closed.Id)
}
