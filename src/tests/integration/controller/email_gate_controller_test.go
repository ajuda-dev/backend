package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
)

func assertForbiddenUnverifiedEmail(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403, recebeu %d", resp.StatusCode)
	}
	body := decodeRestErr(t, resp)
	if body.Message != "email is not verified" {
		t.Errorf("esperava message 'email is not verified', recebeu '%s'", body.Message)
	}
	if body.Err != "forbidden" || body.Code != fiber.StatusForbidden {
		t.Errorf("esperava error forbidden/403, recebeu %s/%d", body.Err, body.Code)
	}
}

func createUnverifiedUser(t *testing.T, email string) *userdomain.UserDomain {
	t.Helper()
	user := createUserWithRole(t, email, userdomain.UserRoleUser)
	clearEmailVerifiedAt(t, user.Id)
	return user
}

func TestUnverifiedEmailForbiddenOnCommunityAndEventWrites(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	unverified := createUnverifiedUser(t, "gate_unverified@ajuda.dev")
	verified := createUserWithRole(t, "gate_verified@ajuda.dev", userdomain.UserRoleUser)
	unverifiedToken := validTokenFor(t, unverified.Id)
	verifiedToken := validTokenFor(t, verified.Id)
	address := createEventAddress(t, "gate_city")

	communityBody := []byte(`{
		"address_id": "` + address.Id + `",
		"name": "Comunidade Gate Unverified",
		"description": "comunidade de teste do gate"
	}`)
	resp, err := doAuthedRequest(app, newCommunityRegisterRequest(communityBody), unverifiedToken)
	if err != nil {
		t.Fatalf("erro ao criar comunidade como não verificado: %v", err)
	}
	assertForbiddenUnverifiedEmail(t, resp)

	communityId := registerCommunityViaApi(t, app, verifiedToken, address.Id, "Comunidade Gate Verified")

	resp, err = doAuthedRequest(app, newJoinCommunityRequest(communityId), unverifiedToken)
	if err != nil {
		t.Fatalf("erro ao entrar na comunidade como não verificado: %v", err)
	}
	assertForbiddenUnverifiedEmail(t, resp)

	verifiedMember := createUserWithRole(t, "gate_verified_member@ajuda.dev", userdomain.UserRoleUser)
	joinCommunityViaApi(t, app, communityId, validTokenFor(t, verifiedMember.Id))

	eventPayload, err := json.Marshal(eventTestRequest{
		OwnerId:     unverified.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento Gate Unverified",
		Description: "evento de teste do gate",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if err != nil {
		t.Fatalf("erro ao montar body do evento: %v", err)
	}
	resp, err = doAuthedRequest(app, newEventRegisterRequest(eventPayload), unverifiedToken)
	if err != nil {
		t.Fatalf("erro ao criar evento como não verificado: %v", err)
	}
	assertForbiddenUnverifiedEmail(t, resp)

	registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     verified.Id,
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento Gate Verified",
		Description: "evento de teste do gate",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
}

func TestUnverifiedEmailCanJoinEventAndAcceptInvitation(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	unverified := createUnverifiedUser(t, "gate_invitee@ajuda.dev")
	owner := createUserWithRole(t, "gate_invite_owner@ajuda.dev", userdomain.UserRoleUser)
	unverifiedToken := validTokenFor(t, unverified.Id)

	communityEvent := euCreateCommunityEvent(t, app, owner, nil)
	resp := euJoinEvent(t, app, communityEvent.Id, "", unverifiedToken)
	if resp.StatusCode != fiber.StatusCreated {
		body := rest_err.RestErr{}
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		t.Fatalf("esperava 201 no join do evento (fora do gate), recebeu %d (body: %+v)", resp.StatusCode, body)
	}
	resp.Body.Close()

	mentoring := euCreateMentoringEvent(t, app, owner)
	resp = euAddParticipant(t, app, mentoring.Id, `{"user_id":"`+unverified.Id+`","role":"MENTEE"}`, validTokenFor(t, owner.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no convite de mentoria, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euUpdateStatus(t, app, mentoring.Id, unverified.Id, eventdomain.StatusConfirmed, unverifiedToken)
	if resp.StatusCode != fiber.StatusOK {
		body := decodeRestErr(t, resp)
		t.Fatalf("esperava 200 ao aceitar convite (fora do gate), recebeu %d (body: %+v)", resp.StatusCode, body)
	}
	row := euDecodeEventUserDto(t, resp)
	if row.Status != eventdomain.StatusConfirmed {
		t.Errorf("esperava CONFIRMED no aceite, recebeu %s", row.Status)
	}
}
