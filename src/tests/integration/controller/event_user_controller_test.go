package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
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

func euUpdateStatus(t *testing.T, app *fiber.App, eventId string, userId string, status string, comment string, token string) *http.Response {
	t.Helper()
	payload := map[string]string{"status": status}
	if comment != "" {
		payload["comment"] = comment
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("erro ao montar body de status: %v", err)
	}
	return euRequest(t, app, http.MethodPut, "/v1/event/"+eventId+"/participants/"+userId+"/status", string(body), token)
}

func euCancelParticipation(t *testing.T, app *fiber.App, eventId string, userId string, token string) *http.Response {
	t.Helper()
	return euRequest(t, app, http.MethodDelete, "/v1/event/"+eventId+"/participants/"+userId, "", token)
}

func euDecodeEventUserDto(t *testing.T, resp *http.Response) eventdto.EventUserDto {
	t.Helper()
	defer resp.Body.Close()
	var row eventdto.EventUserDto
	if err := json.NewDecoder(resp.Body).Decode(&row); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return row
}

func euDecodeParticipants(t *testing.T, resp *http.Response) []eventdto.EventParticipantDto {
	t.Helper()
	defer resp.Body.Close()
	var participants []eventdto.EventParticipantDto
	if err := json.NewDecoder(resp.Body).Decode(&participants); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return participants
}

func euShareEmail(t *testing.T, app *fiber.App, user *userdomain.UserDomain) {
	t.Helper()
	payload := []byte(`{"configVisibility":{"email":{"shareWithCommunity":true}}}`)
	resp := doPutUser(t, app, user.Id, payload, validTokenFor(t, user.Id))
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao compartilhar o email, recebeu %d", resp.StatusCode)
	}
}

func euInviteMentee(t *testing.T, app *fiber.App, eventId string, menteeId string, ownerId string) {
	t.Helper()
	resp := euAddParticipant(t, app, eventId, `{"user_id":"`+menteeId+`","role":"MENTEE"}`, validTokenFor(t, ownerId))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite de MENTEE, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func euFindParticipant(t *testing.T, participants []eventdto.EventParticipantDto, userId string) eventdto.EventParticipantDto {
	t.Helper()
	for _, participant := range participants {
		if participant.UserId == userId {
			return participant
		}
	}
	t.Fatalf("esperava participante %s na listagem", userId)
	return eventdto.EventParticipantDto{}
}

func euFindRow(t *testing.T, eventId string, userId string) evententity.EventUserEntity {
	t.Helper()
	var row evententity.EventUserEntity
	if err := db.Where("event_id = ? AND user_id = ?", eventId, userId).First(&row).Error; err != nil {
		t.Fatalf("esperava achar a linha de participação (%s, %s), recebeu %v", eventId, userId, err)
	}
	return row
}

func euCountRows(t *testing.T, eventId string, userId string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&evententity.EventUserEntity{}).
		Where("event_id = ? AND user_id = ?", eventId, userId).Count(&count).Error; err != nil {
		t.Fatalf("erro ao contar linhas de participação: %v", err)
	}
	return count
}

func euCreateCommunityEvent(t *testing.T, app *fiber.App, owner *userdomain.UserDomain, maxSlots *int) eventdto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento da comunidade",
		Description: "evento de teste de participação",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		MaxSlots:    maxSlots,
	})
}

func euCreateMentoringEvent(t *testing.T, app *fiber.App, owner *userdomain.UserDomain) eventdto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "Mentoria 1:1",
		Description: "evento de mentoria",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
}

func TestJoinEventUsesTokenIdentity(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "join_owner@ajuda.dev", userdomain.UserRoleUser)
	requester := createUserWithRole(t, "join_requester@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "join_third@ajuda.dev", userdomain.UserRoleUser)
	event := euCreateCommunityEvent(t, app, owner, nil)

	resp := euJoinEvent(t, app, event.Id, `{"user_id":"`+third.Id+`"}`, validTokenFor(t, requester.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no join, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.UserId != requester.Id {
		t.Errorf("esperava user_id '%s' (do token), recebeu '%s'", requester.Id, row.UserId)
	}
	if row.Role != eventdomain.RoleAttendee || row.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava ATTENDEE/CONFIRMED, recebeu %s/%s", row.Role, row.Status)
	}
	if euCountRows(t, event.Id, third.Id) != 0 {
		t.Errorf("esperava nenhuma linha para o usuário do body, recebeu %d", euCountRows(t, event.Id, third.Id))
	}
}

func TestJoinEventReactivatesCancelledRow(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "rejoin_owner@ajuda.dev", userdomain.UserRoleUser)
	user := createUserWithRole(t, "rejoin_user@ajuda.dev", userdomain.UserRoleUser)
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
	if cancelledRow.Status != eventdomain.StatusCancelled {
		t.Errorf("esperava CANCELLED após o cancelamento, recebeu %s", cancelledRow.Status)
	}

	second := euJoinEvent(t, app, event.Id, "", token)
	if second.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no rejoin, recebeu %d", second.StatusCode)
	}
	secondRow := euDecodeEventUserDto(t, second)
	if secondRow.Status != eventdomain.StatusConfirmed {
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
	owner := createUserWithRole(t, "full_owner@ajuda.dev", userdomain.UserRoleUser)
	first := createUserWithRole(t, "full_first@ajuda.dev", userdomain.UserRoleUser)
	second := createUserWithRole(t, "full_second@ajuda.dev", userdomain.UserRoleUser)
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
	owner := createUserWithRole(t, "mentoring_owner@ajuda.dev", userdomain.UserRoleUser)
	user := createUserWithRole(t, "mentoring_user@ajuda.dev", userdomain.UserRoleUser)
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
	user := createUserWithRole(t, "join_params@ajuda.dev", userdomain.UserRoleUser)
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
	communityOwner := createUserWithRole(t, "add_community_owner@ajuda.dev", userdomain.UserRoleUser)
	eventOwner := createUserWithRole(t, "add_event_owner@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "add_third@ajuda.dev", userdomain.UserRoleUser)
	speaker := createUserWithRole(t, "add_speaker@ajuda.dev", userdomain.UserRoleUser)
	mentee := createUserWithRole(t, "add_mentee@ajuda.dev", userdomain.UserRoleUser)
	speakerByModerator := createUserWithRole(t, "add_speaker_mod@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "add_moderator@ajuda.dev", userdomain.UserRoleModerator)
	address := createEventAddress(t, "add_city")
	community := createEventCommunity(t, "Comunidade add", communityOwner, address)
	eaAddCommunityMember(t, community.Id, eventOwner.Id)
	eaAddCommunityMember(t, community.Id, third.Id)
	eaAddCommunityMember(t, community.Id, speaker.Id)
	eaAddCommunityMember(t, community.Id, mentee.Id)
	eaAddCommunityMember(t, community.Id, speakerByModerator.Id)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     eventOwner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
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
	if row.Role != eventdomain.RoleSpeaker || row.Status != eventdomain.StatusConfirmed {
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
	if menteeRow.Role != eventdomain.RoleMentee || menteeRow.Status != eventdomain.StatusRequested {
		t.Errorf("esperava MENTEE/REQUESTED no convite de mentoria, recebeu %s/%s", menteeRow.Role, menteeRow.Status)
	}
}

func TestUpdateParticipantStatusOnlyInvited(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "status_owner@ajuda.dev", userdomain.UserRoleUser)
	mentee := createUserWithRole(t, "status_mentee@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "status_third@ajuda.dev", userdomain.UserRoleUser)

	event := euCreateMentoringEvent(t, app, owner)
	resp := euAddParticipant(t, app, event.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusRejected, "", validTokenFor(t, third.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para terceiro recusando sem comment, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 para o criador aceitando pelo convidado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusRejected, "Agenda conflitou nesta semana", validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o convidado recusando, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Status != eventdomain.StatusRejected {
		t.Errorf("esperava REJECTED, recebeu %s", row.Status)
	}
	if row.Comment != "Agenda conflitou nesta semana" {
		t.Errorf("esperava comment na recusa, recebeu '%s'", row.Comment)
	}

	resp = euUpdateStatus(t, app, event.Id, mentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, mentee.Id))
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

	resp = euUpdateStatus(t, app, secondEvent.Id, mentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o convidado aceitando, recebeu %d", resp.StatusCode)
	}
	row = euDecodeEventUserDto(t, resp)
	if row.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava CONFIRMED, recebeu %s", row.Status)
	}

	resp = euUpdateStatus(t, app, secondEvent.Id, mentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na transição CONFIRMED -> CONFIRMED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCancelParticipationPermissions(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "cancel_community_owner@ajuda.dev", userdomain.UserRoleUser)
	eventOwner := createUserWithRole(t, "cancel_event_owner@ajuda.dev", userdomain.UserRoleUser)
	participant := createUserWithRole(t, "cancel_participant@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "cancel_third@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "cancel_moderator@ajuda.dev", userdomain.UserRoleModerator)
	address := createEventAddress(t, "cancel_city")
	community := createEventCommunity(t, "Comunidade cancel", communityOwner, address)
	eaAddCommunityMember(t, community.Id, eventOwner.Id)
	eaAddCommunityMember(t, community.Id, participant.Id)
	eaAddCommunityMember(t, community.Id, third.Id)

	event := registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     eventOwner.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento para cancelamentos",
		Description: "teste de cancelamento",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	eaApproveEvent(t, app, event.Id, communityOwner.Id)
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
			if row.Status != eventdomain.StatusCancelled {
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
	owner := createUserWithRole(t, "keep_owner@ajuda.dev", userdomain.UserRoleUser)
	participant := createUserWithRole(t, "keep_participant@ajuda.dev", userdomain.UserRoleUser)
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
	if row.Status != eventdomain.StatusCancelled {
		t.Errorf("esperava a linha preservada com CANCELLED, recebeu %s", row.Status)
	}

	resp = euGetParticipants(t, app, event.Id, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no GET participants, recebeu %d", resp.StatusCode)
	}
	participants := euDecodeParticipants(t, resp)
	if len(participants) != 1 || participants[0].UserId != participant.Id || participants[0].Status != eventdomain.StatusCancelled {
		t.Errorf("esperava a linha cancelada ainda listada, recebeu %+v", participants)
	}
}

func TestCancelMentoringCreatorCannotLeave(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "mentoring_cancel_owner@ajuda.dev", userdomain.UserRoleUser)
	mentee := createUserWithRole(t, "mentoring_cancel_mentee@ajuda.dev", userdomain.UserRoleUser)
	moderator := createUserWithRole(t, "mentoring_cancel_mod@ajuda.dev", userdomain.UserRoleModerator)
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
	owner := createUserWithRole(t, "email_owner@ajuda.dev", userdomain.UserRoleUser)
	shared := createUserWithRole(t, "email_shared@ajuda.dev", userdomain.UserRoleUser)
	private := createUserWithRole(t, "email_private@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "email_third@ajuda.dev", userdomain.UserRoleUser)
	admin := createUserWithRole(t, "email_admin@ajuda.dev", userdomain.UserRoleAdmin)
	event := euCreateCommunityEvent(t, app, owner, nil)

	for _, user := range []*userdomain.UserDomain{shared, private} {
		resp := euJoinEvent(t, app, event.Id, "", validTokenFor(t, user.Id))
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("esperava 201 no join de preparação, recebeu %d", resp.StatusCode)
		}
		resp.Body.Close()
	}
	euShareEmail(t, app, shared)

	findParticipant := func(participants []eventdto.EventParticipantDto, userId string) eventdto.EventParticipantDto {
		t.Helper()
		for _, participant := range participants {
			if participant.UserId == userId {
				return participant
			}
		}
		t.Fatalf("esperava achar o participante '%s' na listagem, recebeu %+v", userId, participants)
		return eventdto.EventParticipantDto{}
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

func euCreateMentoringEventWithRole(t *testing.T, app *fiber.App, owner *userdomain.UserDomain, creatorRole string) eventdto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "Mentoria 1:1",
		Description: "evento de mentoria",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		CreatorRole: creatorRole,
	})
}

func TestMentoringInviteRoleMustBeComplementary(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	mentorOwner := createUserWithRole(t, "comp_mentor_owner@ajuda.dev", userdomain.UserRoleUser)
	menteeOwner := createUserWithRole(t, "comp_mentee_owner@ajuda.dev", userdomain.UserRoleUser)
	guest := createUserWithRole(t, "comp_guest@ajuda.dev", userdomain.UserRoleUser)

	mentorEvent := euCreateMentoringEvent(t, app, mentorOwner)
	resp := euAddParticipant(t, app, mentorEvent.Id, `{"user_id":"`+guest.Id+`","role":"MENTOR"}`, validTokenFor(t, mentorOwner.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para MENTOR convidando MENTOR, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("role", respBody.Causes)) == 0 {
		t.Errorf("esperava cause em role, recebeu %+v", respBody.Causes)
	}

	resp = euAddParticipant(t, app, mentorEvent.Id, `{"user_id":"`+guest.Id+`","role":"MENTEE"}`, validTokenFor(t, mentorOwner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 para MENTOR convidando MENTEE, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Role != eventdomain.RoleMentee || row.Status != eventdomain.StatusRequested {
		t.Errorf("esperava MENTEE/REQUESTED, recebeu %s/%s", row.Role, row.Status)
	}

	menteeEvent := euCreateMentoringEventWithRole(t, app, menteeOwner, eventdomain.RoleMentee)
	resp = euAddParticipant(t, app, menteeEvent.Id, `{"user_id":"`+guest.Id+`","role":"MENTEE"}`, validTokenFor(t, menteeOwner.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 para MENTEE convidando MENTEE, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	if len(getCauseByField("role", respBody.Causes)) == 0 {
		t.Errorf("esperava cause em role, recebeu %+v", respBody.Causes)
	}

	resp = euAddParticipant(t, app, menteeEvent.Id, `{"user_id":"`+guest.Id+`","role":"MENTOR"}`, validTokenFor(t, menteeOwner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 para MENTEE convidando MENTOR, recebeu %d", resp.StatusCode)
	}
	row = euDecodeEventUserDto(t, resp)
	if row.Role != eventdomain.RoleMentor || row.Status != eventdomain.StatusRequested {
		t.Errorf("esperava MENTOR/REQUESTED, recebeu %s/%s", row.Role, row.Status)
	}
}

func TestMentoringCreatedByMenteeAcceptsMentorInvite(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	mentee := createUserWithRole(t, "by_mentee_owner@ajuda.dev", userdomain.UserRoleUser)
	mentor := createUserWithRole(t, "by_mentee_mentor@ajuda.dev", userdomain.UserRoleUser)

	event := euCreateMentoringEventWithRole(t, app, mentee, eventdomain.RoleMentee)
	if event.MaxSlots == nil || *event.MaxSlots != 2 {
		t.Errorf("esperava max_slots 2 no 1:1, recebeu %+v", event.MaxSlots)
	}
	if event.CreatorRole != eventdomain.RoleMentee {
		t.Errorf("esperava creator_role MENTEE na resposta do register, recebeu '%s'", event.CreatorRole)
	}

	creatorRow := euFindRow(t, event.Id, mentee.Id)
	if creatorRow.Role != eventdomain.RoleMentee || creatorRow.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava a linha do criador MENTEE/CONFIRMED, recebeu %s/%s", creatorRow.Role, creatorRow.Status)
	}

	resp := euAddParticipant(t, app, event.Id, `{"user_id":"`+mentor.Id+`","role":"MENTOR"}`, validTokenFor(t, mentee.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite do mentor, recebeu %d", resp.StatusCode)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Role != eventdomain.RoleMentor || row.Status != eventdomain.StatusRequested {
		t.Errorf("esperava MENTOR/REQUESTED no convite, recebeu %s/%s", row.Role, row.Status)
	}

	resp = euUpdateStatus(t, app, event.Id, mentor.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, mentor.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 para o mentor aceitando, recebeu %d", resp.StatusCode)
	}
	row = euDecodeEventUserDto(t, resp)
	if row.Role != eventdomain.RoleMentor || row.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava MENTOR/CONFIRMED no aceite, recebeu %s/%s", row.Role, row.Status)
	}
}

func TestMentoringCreatorRoleDefaultIsMentor(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "default_role_owner@ajuda.dev", userdomain.UserRoleUser)

	event := euCreateMentoringEvent(t, app, owner)
	if event.CreatorRole != eventdomain.RoleMentor {
		t.Errorf("esperava creator_role MENTOR na resposta do register, recebeu '%s'", event.CreatorRole)
	}
	creatorRow := euFindRow(t, event.Id, owner.Id)
	if creatorRow.Role != eventdomain.RoleMentor || creatorRow.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava a linha do criador MENTOR/CONFIRMED por default, recebeu %s/%s", creatorRow.Role, creatorRow.Status)
	}
}

func TestMentoringCreatorRoleOnlyForMentoring(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "role_scope_owner@ajuda.dev", userdomain.UserRoleUser)

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento com creator_role",
		Description: "creator_role fora de MENTORING",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		CreatorRole: eventdomain.RoleMentee,
	}, "creator_role")

	eventReqValidation(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		Category:    eventdomain.CategoryMentoring,
		Type:        eventdomain.TypeOnline,
		Title:       "Mentoria com papel inválido",
		Description: "creator_role inválido",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
		CreatorRole: "SPEAKER",
	}, "creator_role")
}

func TestMentoringMenteeMaxOneConfirmed(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "max_mentee_owner@ajuda.dev", userdomain.UserRoleUser)
	firstMentee := createUserWithRole(t, "max_mentee_first@ajuda.dev", userdomain.UserRoleUser)
	secondMentee := createUserWithRole(t, "max_mentee_second@ajuda.dev", userdomain.UserRoleUser)

	event := euCreateMentoringEvent(t, app, owner)
	resp := euAddParticipant(t, app, event.Id, `{"user_id":"`+firstMentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite do primeiro mentee, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, firstMentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, firstMentee.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite do primeiro mentee, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euAddParticipant(t, app, event.Id, `{"user_id":"`+secondMentee.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite do segundo mentee, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, event.Id, secondMentee.Id, eventdomain.StatusConfirmed, "", validTokenFor(t, secondMentee.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no segundo mentee confirmado, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	if len(getCauseByField("user_id", respBody.Causes)) == 0 {
		t.Errorf("esperava cause em user_id, recebeu %+v", respBody.Causes)
	}
}

func TestUpdateParticipantStatusComment(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	owner := createUserWithRole(t, "comment_owner@ajuda.dev", userdomain.UserRoleUser)
	mentee := createUserWithRole(t, "comment_mentee@ajuda.dev", userdomain.UserRoleUser)
	menteeToken := validTokenFor(t, mentee.Id)
	ownerToken := validTokenFor(t, owner.Id)
	tooLong := strings.Repeat("a", 501)

	validationEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, validationEvent.Id, mentee.Id, owner.Id)

	resp := euUpdateStatus(t, app, validationEvent.Id, mentee.Id, eventdomain.StatusRejected, "", menteeToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 ao recusar sem comment, recebeu %d", resp.StatusCode)
	}
	respBody := decodeRestErr(t, resp)
	causes := getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment is required when rejecting" {
		t.Errorf("esperava cause comment is required when rejecting, recebeu %+v", respBody.Causes)
	}

	resp = euRequest(t, app, http.MethodPut, "/v1/event/"+validationEvent.Id+"/participants/"+mentee.Id+"/status",
		`{"status":"REJECTED","comment":""}`, menteeToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 ao recusar com comment vazio, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes = getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment is required when rejecting" {
		t.Errorf("esperava cause comment is required when rejecting, recebeu %+v", respBody.Causes)
	}

	resp = euUpdateStatus(t, app, validationEvent.Id, mentee.Id, eventdomain.StatusRejected, "   ", menteeToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 ao recusar com comment só de espaços, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes = getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment is required when rejecting" {
		t.Errorf("esperava cause comment is required when rejecting, recebeu %+v", respBody.Causes)
	}

	resp = euUpdateStatus(t, app, validationEvent.Id, mentee.Id, eventdomain.StatusRejected, tooLong, menteeToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no comment > 500 em REJECTED, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes = getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment must have at most 500 characters" {
		t.Errorf("esperava cause comment must have at most 500 characters, recebeu %+v", respBody.Causes)
	}

	resp = euUpdateStatus(t, app, validationEvent.Id, mentee.Id, eventdomain.StatusConfirmed, tooLong, menteeToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no comment > 500 em CONFIRMED, recebeu %d", resp.StatusCode)
	}
	respBody = decodeRestErr(t, resp)
	causes = getCauseByField("comment", respBody.Causes)
	if len(causes) == 0 || causes[0] != "comment must have at most 500 characters" {
		t.Errorf("esperava cause comment must have at most 500 characters, recebeu %+v", respBody.Causes)
	}

	resp = euUpdateStatus(t, app, validationEvent.Id, mentee.Id, eventdomain.StatusConfirmed, "", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite sem comment, recebeu %d", resp.StatusCode)
	}
	acceptBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("erro ao ler body do aceite: %v", err)
	}
	var accepted eventdto.EventUserDto
	if err := json.Unmarshal(acceptBody, &accepted); err != nil {
		t.Fatalf("erro ao decodificar aceite: %v", err)
	}
	if accepted.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava CONFIRMED, recebeu %s", accepted.Status)
	}
	if accepted.Comment != "" {
		t.Errorf("não esperava comment no aceite sem texto, recebeu '%s'", accepted.Comment)
	}
	var acceptedRaw map[string]interface{}
	if err := json.Unmarshal(acceptBody, &acceptedRaw); err != nil {
		t.Fatalf("erro ao decodificar aceite raw: %v", err)
	}
	if _, ok := acceptedRaw["comment"]; ok {
		t.Errorf("não esperava chave comment no aceite sem texto")
	}
	dbRow := euFindRow(t, validationEvent.Id, mentee.Id)
	if dbRow.StatusComment != nil {
		t.Errorf("esperava status_comment NULL no aceite sem texto, recebeu %q", *dbRow.StatusComment)
	}
	resp = euGetParticipants(t, app, validationEvent.Id, ownerToken)
	listed := euDecodeParticipants(t, resp)
	if euFindParticipant(t, listed, mentee.Id).Comment != "" {
		t.Errorf("não esperava comment na listagem após aceite sem texto")
	}

	rejectEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, rejectEvent.Id, mentee.Id, owner.Id)
	resp = euUpdateStatus(t, app, rejectEvent.Id, mentee.Id, eventdomain.StatusRejected, "Agenda conflitou nesta semana", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na recusa com comment, recebeu %d", resp.StatusCode)
	}
	rejected := euDecodeEventUserDto(t, resp)
	if rejected.Comment != "Agenda conflitou nesta semana" {
		t.Errorf("esperava o comment da recusa, recebeu '%s'", rejected.Comment)
	}
	resp = euGetParticipants(t, app, rejectEvent.Id, ownerToken)
	listed = euDecodeParticipants(t, resp)
	if got := euFindParticipant(t, listed, mentee.Id).Comment; got != "Agenda conflitou nesta semana" {
		t.Errorf("esperava o comment na listagem, recebeu '%s'", got)
	}

	acceptEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, acceptEvent.Id, mentee.Id, owner.Id)
	resp = euUpdateStatus(t, app, acceptEvent.Id, mentee.Id, eventdomain.StatusConfirmed, "Nos falamos pelo LinkedIn", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite com comment, recebeu %d", resp.StatusCode)
	}
	confirmed := euDecodeEventUserDto(t, resp)
	if confirmed.Comment != "Nos falamos pelo LinkedIn" {
		t.Errorf("esperava o comment do aceite, recebeu '%s'", confirmed.Comment)
	}
	resp = euGetParticipants(t, app, acceptEvent.Id, ownerToken)
	listed = euDecodeParticipants(t, resp)
	if got := euFindParticipant(t, listed, mentee.Id).Comment; got != "Nos falamos pelo LinkedIn" {
		t.Errorf("esperava o comment na listagem do aceite, recebeu '%s'", got)
	}

	trimEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, trimEvent.Id, mentee.Id, owner.Id)
	resp = euUpdateStatus(t, app, trimEvent.Id, mentee.Id, eventdomain.StatusRejected, "  motivo  ", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na recusa com trim, recebeu %d", resp.StatusCode)
	}
	trimmed := euDecodeEventUserDto(t, resp)
	if trimmed.Comment != "motivo" {
		t.Errorf("esperava comment trimado 'motivo', recebeu '%s'", trimmed.Comment)
	}
	dbRow = euFindRow(t, trimEvent.Id, mentee.Id)
	if dbRow.StatusComment == nil || *dbRow.StatusComment != "motivo" {
		t.Errorf("esperava status_comment 'motivo', recebeu %+v", dbRow.StatusComment)
	}

	cancelEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, cancelEvent.Id, mentee.Id, owner.Id)
	resp = euUpdateStatus(t, app, cancelEvent.Id, mentee.Id, eventdomain.StatusConfirmed, "Nos falamos pelo LinkedIn", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no aceite antes do cancel, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = euCancelParticipation(t, app, cancelEvent.Id, mentee.Id, menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no cancel após aceite com comment, recebeu %d", resp.StatusCode)
	}
	cancelled := euDecodeEventUserDto(t, resp)
	if cancelled.Comment != "" {
		t.Errorf("não esperava comment após o cancel, recebeu '%s'", cancelled.Comment)
	}
	dbRow = euFindRow(t, cancelEvent.Id, mentee.Id)
	if dbRow.StatusComment != nil {
		t.Errorf("esperava status_comment NULL após o cancel, recebeu %q", *dbRow.StatusComment)
	}
	resp = euGetParticipants(t, app, cancelEvent.Id, ownerToken)
	listed = euDecodeParticipants(t, resp)
	if euFindParticipant(t, listed, mentee.Id).Comment != "" {
		t.Errorf("não esperava comment na listagem após o cancel")
	}

	reinviteEvent := euCreateMentoringEvent(t, app, owner)
	euInviteMentee(t, app, reinviteEvent.Id, mentee.Id, owner.Id)
	resp = euUpdateStatus(t, app, reinviteEvent.Id, mentee.Id, eventdomain.StatusRejected, "Agenda conflitou nesta semana", menteeToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na recusa antes do reconvite, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = euAddParticipant(t, app, reinviteEvent.Id, `{"user_id":"`+mentee.Id+`","role":"MENTEE"}`, ownerToken)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no reconvite após reject, recebeu %d", resp.StatusCode)
	}
	reinvited := euDecodeEventUserDto(t, resp)
	if reinvited.Status != eventdomain.StatusRequested {
		t.Errorf("esperava REQUESTED no reconvite, recebeu %s", reinvited.Status)
	}
	if reinvited.Comment != "" {
		t.Errorf("não esperava comment residual no reconvite, recebeu '%s'", reinvited.Comment)
	}
	dbRow = euFindRow(t, reinviteEvent.Id, mentee.Id)
	if dbRow.StatusComment != nil {
		t.Errorf("esperava status_comment NULL após o reconvite, recebeu %q", *dbRow.StatusComment)
	}
	resp = euGetParticipants(t, app, reinviteEvent.Id, ownerToken)
	listed = euDecodeParticipants(t, resp)
	participant := euFindParticipant(t, listed, mentee.Id)
	if participant.Status != eventdomain.StatusRequested {
		t.Errorf("esperava REQUESTED na listagem após reconvite, recebeu %s", participant.Status)
	}
	if participant.Comment != "" {
		t.Errorf("não esperava comment na listagem após o reconvite")
	}
}
