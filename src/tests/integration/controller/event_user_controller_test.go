package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func euRequest(t *testing.T, app *fiber.App, method string, url string, body string, token string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, url, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	return resp
}

func euJoinEvent(t *testing.T, app *fiber.App, eventId string, body string, token string) *http.Response {
	t.Helper()
	return euRequest(t, app, http.MethodPost, "/v1/event/"+eventId+"/join", body, token)
}

func euAddParticipant(t *testing.T, app *fiber.App, eventId string, body string, token string) *http.Response {
	t.Helper()
	return euRequest(t, app, http.MethodPost, "/v1/event/"+eventId+"/participants", body, token)
}

func euGetParticipants(t *testing.T, app *fiber.App, eventId string, token string) *http.Response {
	t.Helper()
	return euRequest(t, app, http.MethodGet, "/v1/event/"+eventId+"/participants", "", token)
}

func euUpdateStatus(t *testing.T, app *fiber.App, eventId string, userId string, status string, token string) *http.Response {
	t.Helper()
	body := `{"status":"` + status + `"}`
	return euRequest(t, app, http.MethodPut, "/v1/event/"+eventId+"/participants/"+userId+"/status", body, token)
}

func euCancelParticipation(t *testing.T, app *fiber.App, eventId string, userId string, token string) *http.Response {
	t.Helper()
	return euRequest(t, app, http.MethodDelete, "/v1/event/"+eventId+"/participants/"+userId, "", token)
}

func euDecodeEventUserDto(t *testing.T, resp *http.Response) dto.EventUserDto {
	t.Helper()
	defer resp.Body.Close()
	var row dto.EventUserDto
	if err := json.NewDecoder(resp.Body).Decode(&row); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return row
}

func euDecodeParticipants(t *testing.T, resp *http.Response) []dto.EventParticipantDto {
	t.Helper()
	defer resp.Body.Close()
	var participants []dto.EventParticipantDto
	if err := json.NewDecoder(resp.Body).Decode(&participants); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return participants
}

func euShareEmail(t *testing.T, app *fiber.App, user *domain.UserDomain) {
	t.Helper()
	payload := []byte(`{"configVisibility":{"email":{"shareWithCommunity":true}}}`)
	resp := doPutUser(t, app, user.Id, payload, validTokenFor(t, user.Id))
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao compartilhar o email, recebeu %d", resp.StatusCode)
	}
}

func euFindRow(t *testing.T, eventId string, userId string) entity.EventUserEntity {
	t.Helper()
	var row entity.EventUserEntity
	if err := db.Where("event_id = ? AND user_id = ?", eventId, userId).First(&row).Error; err != nil {
		t.Fatalf("esperava achar a linha de participação (%s, %s), recebeu %v", eventId, userId, err)
	}
	return row
}

func euCountRows(t *testing.T, eventId string, userId string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&entity.EventUserEntity{}).
		Where("event_id = ? AND user_id = ?", eventId, userId).Count(&count).Error; err != nil {
		t.Fatalf("erro ao contar linhas de participação: %v", err)
	}
	return count
}

func euCreateCommunityEvent(t *testing.T, app *fiber.App, owner *domain.UserDomain, maxSlots *int) dto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento da comunidade",
		Description: "evento de teste de participação",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		MaxSlots:    maxSlots,
	})
}

func euCreateMentoringEvent(t *testing.T, app *fiber.App, owner *domain.UserDomain) dto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    domain.CategoryMentoring,
		Type:        domain.TypeOnline,
		Title:       "Mentoria 1:1",
		Description: "evento de mentoria",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
}

func TestJoinEventUsesTokenIdentity(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "join_owner@ajuda.dev", domain.UserRoleUser)
	requester := createUserWithRole(t, "join_requester@ajuda.dev", domain.UserRoleUser)
	third := createUserWithRole(t, "join_third@ajuda.dev", domain.UserRoleUser)
	event := euCreateCommunityEvent(t, app, owner, nil)

	resp := euJoinEvent(t, app, event.Id, `{"user_id":"`+third.Id+`"}`, validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.UserId != requester.Id {
		t.Errorf("esperava user_id '%s' (do token), recebeu '%s'", requester.Id, row.UserId)
	}
	if row.Role != domain.RoleAttendee || row.Status != domain.StatusConfirmed {
		t.Errorf("esperava ATTENDEE/CONFIRMED, recebeu %s/%s", row.Role, row.Status)
	}
	if euCountRows(t, event.Id, third.Id) != 0 {
		t.Errorf("esperava nenhuma linha para o usuário do body, recebeu %d", euCountRows(t, event.Id, third.Id))
	}
}

func TestJoinEventReactivatesCancelledRow(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "rejoin_owner@ajuda.dev", domain.UserRoleUser)
	user := createUserWithRole(t, "rejoin_user@ajuda.dev", domain.UserRoleUser)
	token := validTokenFor(t, user.Id)
	event := euCreateCommunityEvent(t, app, owner, nil)

	first := euJoinEvent(t, app, event.Id, "", token)
	if first.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no primeiro join, recebeu %d", first.StatusCode)
	}
	firstRow := euDecodeEventUserDto(t, first)

	cancel := euCancelParticipation(t, app, event.Id, user.Id, token)
	if cancel.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no cancelamento, recebeu %d", cancel.StatusCode)
	}
	cancelledRow := euDecodeEventUserDto(t, cancel)
	if cancelledRow.Status != domain.StatusCancelled {
		t.Errorf("esperava CANCELLED após o cancelamento, recebeu %s", cancelledRow.Status)
	}

	second := euJoinEvent(t, app, event.Id, "", token)
	if second.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no rejoin, recebeu %d", second.StatusCode)
	}
	secondRow := euDecodeEventUserDto(t, second)
	if secondRow.Status != domain.StatusConfirmed {
		t.Errorf("esperava CONFIRMED após o rejoin, recebeu %s", secondRow.Status)
	}
	if secondRow.Id != firstRow.Id {
		t.Errorf("esperava a mesma linha reativada ('%s'), recebeu '%s'", firstRow.Id, secondRow.Id)
	}
	if euCountRows(t, event.Id, user.Id) != 1 {
		t.Errorf("esperava uma única linha por (evento, usuário), recebeu %d", euCountRows(t, event.Id, user.Id))
	}
}

func TestJoinEventFull(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "full_owner@ajuda.dev", domain.UserRoleUser)
	first := createUserWithRole(t, "full_first@ajuda.dev", domain.UserRoleUser)
	second := createUserWithRole(t, "full_second@ajuda.dev", domain.UserRoleUser)
	maxSlots := 1
	event := euCreateCommunityEvent(t, app, owner, &maxSlots)

	resp := euJoinEvent(t, app, event.Id, "", validTokenFor(t, first.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no primeiro join, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euJoinEvent(t, app, event.Id, "", validTokenFor(t, second.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no join sem vaga, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("event_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "event is full" {
		t.Errorf("esperava a cause 'event is full' em event_id, recebeu %+v", respBody.Causes)
	}
}

func TestJoinMentoringEventBlocked(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "mentoring_owner@ajuda.dev", domain.UserRoleUser)
	user := createUserWithRole(t, "mentoring_user@ajuda.dev", domain.UserRoleUser)
	event := euCreateMentoringEvent(t, app, owner)

	resp := euJoinEvent(t, app, event.Id, "", validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no join de MENTORING, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("event_id", respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava cause em event_id para MENTORING, recebeu %+v", respBody.Causes)
	}
}

func TestJoinEventInvalidParams(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	user := createUserWithRole(t, "join_params@ajuda.dev", domain.UserRoleUser)
	token := validTokenFor(t, user.Id)

	resp := euJoinEvent(t, app, "invalid-id", "", token)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 para uuid inválido, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euJoinEvent(t, app, uuidv7.New().String(), "", token)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 para evento inexistente, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euJoinEvent(t, app, uuidv7.New().String(), "", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAddParticipantOnlyManager(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "add_community_owner@ajuda.dev", domain.UserRoleUser)
	eventOwner := createUserWithRole(t, "add_event_owner@ajuda.dev", domain.UserRoleUser)
	third := createUserWithRole(t, "add_third@ajuda.dev", domain.UserRoleUser)
	speaker := createUserWithRole(t, "add_speaker@ajuda.dev", domain.UserRoleUser)
	mentee := createUserWithRole(t, "add_mentee@ajuda.dev", domain.UserRoleUser)
	speakerByModerator := createUserWithRole(t, "add_speaker_mod@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "add_moderator@ajuda.dev", domain.UserRoleModerator)
	address := createEventAddress(t, "add_city")
	community := createEventCommunity(t, "Comunidade add", communityOwner, address)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     eventOwner.Id,
		CommunityId: strPtr(community.Id),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento com convidados",
		Description: "teste de convite",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})

	resp := euAddParticipant(t, app, event.Id, `{"user_id":"`+speaker.Id+`","role":"SPEAKER"}`, validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+speaker.Id+`","role":"SPEAKER"}`, validTokenFor(t, eventOwner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 para o criador no convite, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Role != domain.RoleSpeaker || row.Status != domain.StatusConfirmed {
		t.Errorf("esperava SPEAKER/CONFIRMED no convite, recebeu %s/%s", row.Role, row.Status)
	}

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+mentee.Id+`","role":"SPEAKER"}`, validTokenFor(t, communityOwner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 para o dono da comunidade no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+speakerByModerator.Id+`","role":"SPEAKER"}`, validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 para moderador no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	mentoring := euCreateMentoringEvent(t, app, eventOwner)
	resp = euAddParticipant(t, app, mentoring.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, validTokenFor(t, eventOwner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite de MENTEE, recebeu %d", resp.StatusCode)
	}
	menteeRow := euDecodeEventUserDto(t, resp)
	if menteeRow.Role != domain.RoleMentee || menteeRow.Status != domain.StatusRequested {
		t.Errorf("esperava MENTEE/REQUESTED no convite de mentoria, recebeu %s/%s", menteeRow.Role, menteeRow.Status)
	}
}

func TestUpdateParticipantStatusOnlyInvited(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "status_owner@ajuda.dev", domain.UserRoleUser)
	mentee := createUserWithRole(t, "status_mentee@ajuda.dev", domain.UserRoleUser)
	third := createUserWithRole(t, "status_third@ajuda.dev", domain.UserRoleUser)

	event := euCreateMentoringEvent(t, app, owner)
	resp := euAddParticipant(t, app, event.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, domain.StatusConfirmed, validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, domain.StatusConfirmed, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para o criador aceitando pelo convidado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, domain.StatusRejected, validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o convidado recusando, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Status != domain.StatusRejected {
		t.Errorf("esperava REJECTED, recebeu %s", row.Status)
	}

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, domain.StatusConfirmed, validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na transição REJECTED -> CONFIRMED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	secondEvent := euCreateMentoringEvent(t, app, owner)
	resp = euAddParticipant(t, app, secondEvent.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite do segundo evento, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, secondEvent.Id, mentee.Id, domain.StatusConfirmed, validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o convidado aceitando, recebeu %d", resp.StatusCode)
	}
	row = euDecodeEventUserDto(t, resp)
	if row.Status != domain.StatusConfirmed {
		t.Errorf("esperava CONFIRMED, recebeu %s", row.Status)
	}

	resp = euUpdateStatus(t, app, secondEvent.Id, mentee.Id, domain.StatusConfirmed, validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na transição CONFIRMED -> CONFIRMED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCancelParticipationPermissions(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "cancel_community_owner@ajuda.dev", domain.UserRoleUser)
	eventOwner := createUserWithRole(t, "cancel_event_owner@ajuda.dev", domain.UserRoleUser)
	participant := createUserWithRole(t, "cancel_participant@ajuda.dev", domain.UserRoleUser)
	third := createUserWithRole(t, "cancel_third@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "cancel_moderator@ajuda.dev", domain.UserRoleModerator)
	address := createEventAddress(t, "cancel_city")
	community := createEventCommunity(t, "Comunidade cancel", communityOwner, address)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     eventOwner.Id,
		CommunityId: strPtr(community.Id),
		Category:    domain.CategoryCommunityEvent,
		Type:        domain.TypeOnline,
		Title:       "Evento para cancelamentos",
		Description: "teste de cancelamento",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	participantToken := validTokenFor(t, participant.Id)

	cases := []struct {
		name   string
		token  string
		expect int
	}{
		{name: "terceiro", token: validTokenFor(t, third.Id), expect: fiber.StatusForbidden},
		{name: "self", token: participantToken, expect: fiber.StatusOK},
		{name: "criador do evento", token: validTokenFor(t, eventOwner.Id), expect: fiber.StatusOK},
		{name: "dono da comunidade", token: validTokenFor(t, communityOwner.Id), expect: fiber.StatusOK},
		{name: "moderador", token: validTokenFor(t, moderator.Id), expect: fiber.StatusOK},
	}

	for _, testCase := range cases {
		reset := euCancelParticipation(t, app, event.Id, participant.Id, participantToken)
		reset.Body.Close()

		resp := euJoinEvent(t, app, event.Id, "", participantToken)
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("[%s] esperava 201 no join de preparação, recebeu %d", testCase.name, resp.StatusCode)
		}
		resp.Body.Close()

		resp = euCancelParticipation(t, app, event.Id, participant.Id, testCase.token)
		if resp.StatusCode != testCase.expect {
			t.Errorf("[%s] esperava %d no cancelamento, recebeu %d", testCase.name, testCase.expect, resp.StatusCode)
		}
		if resp.StatusCode == fiber.StatusOK {
			row := euDecodeEventUserDto(t, resp)
			if row.Status != domain.StatusCancelled {
				t.Errorf("[%s] esperava CANCELLED, recebeu %s", testCase.name, row.Status)
			}
		} else {
			resp.Body.Close()
		}
	}
}

func TestCancelParticipationKeepsRow(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "keep_owner@ajuda.dev", domain.UserRoleUser)
	participant := createUserWithRole(t, "keep_participant@ajuda.dev", domain.UserRoleUser)
	participantToken := validTokenFor(t, participant.Id)
	event := euCreateCommunityEvent(t, app, owner, nil)

	resp := euJoinEvent(t, app, event.Id, "", participantToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euCancelParticipation(t, app, event.Id, participant.Id, participantToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no cancelamento, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	row := euFindRow(t, event.Id, participant.Id)
	if row.Status != domain.StatusCancelled {
		t.Errorf("esperava a linha preservada com CANCELLED, recebeu %s", row.Status)
	}

	resp = euGetParticipants(t, app, event.Id, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET participants, recebeu %d", resp.StatusCode)
	}
	participants := euDecodeParticipants(t, resp)
	if len(participants) != 1 || participants[0].UserId != participant.Id || participants[0].Status != domain.StatusCancelled {
		t.Errorf("esperava a linha cancelada ainda listada, recebeu %+v", participants)
	}
}

func TestCancelMentoringCreatorCannotLeave(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "mentoring_cancel_owner@ajuda.dev", domain.UserRoleUser)
	mentee := createUserWithRole(t, "mentoring_cancel_mentee@ajuda.dev", domain.UserRoleUser)
	moderator := createUserWithRole(t, "mentoring_cancel_mod@ajuda.dev", domain.UserRoleModerator)
	event := euCreateMentoringEvent(t, app, owner)

	resp := euCancelParticipation(t, app, event.Id, owner.Id, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para o criador saindo do 1:1, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("user_id", respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava cause em user_id, recebeu %+v", respBody.Causes)
	}

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euCancelParticipation(t, app, event.Id, mentee.Id, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 para o criador cancelando a linha do convidado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euCancelParticipation(t, app, event.Id, owner.Id, validTokenFor(t, moderator.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 para o moderador cancelando a linha do criador no 1:1, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetParticipantsEmailFiltered(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "email_owner@ajuda.dev", domain.UserRoleUser)
	shared := createUserWithRole(t, "email_shared@ajuda.dev", domain.UserRoleUser)
	private := createUserWithRole(t, "email_private@ajuda.dev", domain.UserRoleUser)
	third := createUserWithRole(t, "email_third@ajuda.dev", domain.UserRoleUser)
	admin := createUserWithRole(t, "email_admin@ajuda.dev", domain.UserRoleAdmin)
	event := euCreateCommunityEvent(t, app, owner, nil)

	for _, user := range []*domain.UserDomain{shared, private} {
		resp := euJoinEvent(t, app, event.Id, "", validTokenFor(t, user.Id))
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("esperava 201 no join de preparação, recebeu %d", resp.StatusCode)
		}
		resp.Body.Close()
	}
	euShareEmail(t, app, shared)

	findParticipant := func(participants []dto.EventParticipantDto, userId string) dto.EventParticipantDto {
		t.Helper()
		for _, participant := range participants {
			if participant.UserId == userId {
				return participant
			}
		}
		t.Fatalf("esperava achar o participante '%s' na listagem, recebeu %+v", userId, participants)
		return dto.EventParticipantDto{}
	}

	resp := euGetParticipants(t, app, event.Id, validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET participants, recebeu %d", resp.StatusCode)
	}
	participants := euDecodeParticipants(t, resp)
	if len(participants) != 2 {
		t.Fatalf("esperava 2 participantes, recebeu %d", len(participants))
	}
	sharedRow := findParticipant(participants, shared.Id)
	if sharedRow.User == nil || sharedRow.User.Email != shared.Email {
		t.Errorf("esperava o email compartilhado visível ao terceiro, recebeu %+v", sharedRow.User)
	}
	privateRow := findParticipant(participants, private.Id)
	if privateRow.User == nil || privateRow.User.Email != "" {
		t.Errorf("esperava o email não compartilhado omitido, recebeu %+v", privateRow.User)
	}
	if privateRow.User != nil && privateRow.User.Name != private.Name {
		t.Errorf("esperava o nome do participante, recebeu '%s'", privateRow.User.Name)
	}

	resp = euGetParticipants(t, app, event.Id, validTokenFor(t, private.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET participants do próprio usuário, recebeu %d", resp.StatusCode)
	}
	participants = euDecodeParticipants(t, resp)
	if selfRow := findParticipant(participants, private.Id); selfRow.User == nil || selfRow.User.Email != private.Email {
		t.Errorf("esperava o próprio email completo, recebeu %+v", selfRow.User)
	}

	resp = euGetParticipants(t, app, event.Id, validTokenFor(t, admin.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET participants do admin, recebeu %d", resp.StatusCode)
	}
	participants = euDecodeParticipants(t, resp)
	if adminRow := findParticipant(participants, private.Id); adminRow.User == nil || adminRow.User.Email != private.Email {
		t.Errorf("esperava o email completo para o admin, recebeu %+v", adminRow.User)
	}
}
