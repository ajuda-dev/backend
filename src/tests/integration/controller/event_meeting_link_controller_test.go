package controller_test

import (
	"net/http"
	"strings"
	"testing"

	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

const laterMeetingLink = "https://meet.google.com/later-room"

func TestOwnerCanSetMeetingLinkAfterCreate(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_owner@ajuda.dev", userdomain.UserRoleUser)
	attendee := createUserWithRole(t, "meeting_link_attendee@ajuda.dev", userdomain.UserRoleUser)
	ownerToken := validTokenFor(t, owner.Id)
	attendeeToken := validTokenFor(t, attendee.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar sem link",
		Description: "link depois",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
	})
	if created.MeetingLink != "" {
		t.Fatalf("esperava meeting_link vazio no create, recebeu %q", created.MeetingLink)
	}

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":"`+laterMeetingLink+`"}`, ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no PUT meeting-link do owner, recebeu %d", resp.StatusCode)
	}
	_, ownerRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, ownerRaw, laterMeetingLink)

	resp = euJoinEvent(t, app, created.Id, "", attendeeToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+created.Id, "", attendeeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no detalhe do participante, recebeu %d", resp.StatusCode)
	}
	_, attendeeRaw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, attendeeRaw, laterMeetingLink)
}

func TestHybridEventAcceptsMeetingLinkUpdate(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_hybrid_owner@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "Campinas")
	ownerToken := validTokenFor(t, owner.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		AddressId:   strPtr(address.Id),
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeHybrid,
		Title:       "Híbrido sem link",
		Description: "sala depois",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
	})

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":"`+laterMeetingLink+`"}`, ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no PUT meeting-link HYBRID, recebeu %d", resp.StatusCode)
	}
	_, raw := decodeEventRaw(t, resp)
	assertMeetingLinkPresent(t, raw, laterMeetingLink)
}

func TestInpersonEventRejectsMeetingLinkUpdate(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_inperson_owner@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "Santos")
	ownerToken := validTokenFor(t, owner.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		AddressId:   strPtr(address.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
		Title:       "Presencial",
		Description: "sem sala",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
	})

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":"`+laterMeetingLink+`"}`, ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no PUT meeting-link INPERSON, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("meeting_link", respBody.Causes)
	if len(causes) == 0 || causes[0] != "meeting_link is only allowed for ONLINE and HYBRID events" {
		t.Errorf("esperava cause de INPERSON, recebeu %+v", respBody.Causes)
	}
}

func TestMeetingLinkUpdateRejectsInvalidURL(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_invalid_owner@ajuda.dev", userdomain.UserRoleUser)
	ownerToken := validTokenFor(t, owner.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar url inválida",
		Description: "url",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
	})

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":"not-a-url"}`, ownerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para URL inválida, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("meeting_link", respBody.Causes)
	if len(causes) == 0 || causes[0] != "meeting_link must be a valid http or https url" {
		t.Errorf("esperava cause de URL inválida, recebeu %+v", respBody.Causes)
	}
}

func TestThirdPartyCannotUpdateMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_403_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "meeting_link_403_third@ajuda.dev", userdomain.UserRoleUser)
	thirdToken := validTokenFor(t, third.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar alheio",
		Description: "terceiro",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
	})

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":"`+laterMeetingLink+`"}`, thirdToken)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if respBody.Message != "only the event owner, the community owner or moderators can manage this event" {
		t.Errorf("esperava 403 de canManageEvent, recebeu '%s'", respBody.Message)
	}
}

func TestEmptyMeetingLinkClearsField(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_clear_owner@ajuda.dev", userdomain.UserRoleUser)
	ownerToken := validTokenFor(t, owner.Id)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar com link",
		Description: "limpar depois",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: laterMeetingLink,
	})
	if created.MeetingLink != laterMeetingLink {
		t.Fatalf("esperava meeting_link no create, recebeu %q", created.MeetingLink)
	}

	resp := euRequest(t, app, http.MethodPut, "/v1/event/"+created.Id+"/meeting-link",
		`{"meeting_link":""}`, ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao limpar meeting_link, recebeu %d", resp.StatusCode)
	}
	_, putRaw := decodeEventRaw(t, resp)
	assertMeetingLinkOmitted(t, putRaw)

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+created.Id, "", ownerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET após limpar, recebeu %d", resp.StatusCode)
	}
	_, getRaw := decodeEventRaw(t, resp)
	assertMeetingLinkOmitted(t, getRaw)
}

func TestCreateOnlinePersistsTrimmedMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_create_trim@ajuda.dev", userdomain.UserRoleUser)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar com link",
		Description: "url válida com espaço",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: "  https://meet.example.com/x  ",
	})
	if created.MeetingLink != "https://meet.example.com/x" {
		t.Fatalf("esperava meeting_link trimado, recebeu %q", created.MeetingLink)
	}
}

func TestCreateOnlineOmitsEmptyMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_create_empty@ajuda.dev", userdomain.UserRoleUser)

	created := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryWebinar,
		Type:        eventdomain.TypeOnline,
		Title:       "Webinar sem link",
		Description: "campo vazio",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: "",
	})
	if created.MeetingLink != "" {
		t.Fatalf("esperava meeting_link vazio, recebeu %q", created.MeetingLink)
	}
}

func TestCreateRejectsInvalidMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_create_invalid@ajuda.dev", userdomain.UserRoleUser)
	tooLong := "https://exemplo.com/" + strings.Repeat("a", 501-len("https://exemplo.com/"))

	cases := []struct {
		name string
		link string
		want string
	}{
		{"javascript", "javascript:alert(1)", "meeting_link must be a valid http or https url"},
		{"data", "data:text/html,hi", "meeting_link must be a valid http or https url"},
		{"ftp", "ftp://files.example.com/a", "meeting_link must be a valid http or https url"},
		{"sem host", "https://", "meeting_link must be a valid http or https url"},
		{"acima de 500", tooLong, "meeting_link must have at most 500 characters"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			respBody := eventReqValidation(t, app, eventTestRequest{
				OwnerId:     owner.Id,
				Category:    eventdomain.CategoryWebinar,
				Type:        eventdomain.TypeOnline,
				Title:       "Webinar url ruim",
				Description: "create",
				StartAt:     uniqueEventStartAt(),
				DurationMin: 60,
				MeetingLink: tc.link,
			}, "meeting_link")
			causes := getCauseByField("meeting_link", respBody.Causes)
			if len(causes) == 0 || causes[0] != tc.want {
				t.Errorf("esperava %q, recebeu %+v", tc.want, respBody.Causes)
			}
		})
	}
}

func TestCreateInpersonRejectsMeetingLink(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "meeting_link_create_inperson@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "Osasco")

	respBody := eventReqValidation(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		AddressId:   strPtr(address.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
		Title:       "Presencial com link",
		Description: "não deveria",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: laterMeetingLink,
	}, "meeting_link")
	causes := getCauseByField("meeting_link", respBody.Causes)
	if len(causes) == 0 || causes[0] != "meeting_link is only allowed for ONLINE and HYBRID events" {
		t.Errorf("esperava cause de INPERSON no create, recebeu %+v", respBody.Causes)
	}

	respBody = eventReqValidation(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		AddressId:   strPtr(address.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeInperson,
		Title:       "Presencial javascript",
		Description: "não deveria",
		StartAt:     uniqueEventStartAt(),
		DurationMin: 60,
		MeetingLink: "javascript:alert(1)",
	}, "meeting_link")
	causes = getCauseByField("meeting_link", respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava 400 para javascript: em INPERSON, recebeu %+v", respBody.Causes)
	}
}
