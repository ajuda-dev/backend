package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func eaUpdateApproval(t *testing.T, app *fiber.App, eventId string, status string, token string) *http.Response {
	t.Helper()
	payload, err := json.Marshal(eventdto.UpdateEventApprovalDto{Status: status})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	return euRequest(t, app, http.MethodPut, "/v1/event/"+eventId+"/approval", string(payload), token)
}

func eaEventStatus(t *testing.T, eventId string) string {
	t.Helper()
	var row evententity.EventEntity
	if err := db.Where("id = ?", eventId).First(&row).Error; err != nil {
		t.Fatalf("esperava achar o evento %s, recebeu %v", eventId, err)
	}
	return row.Status
}

func eaApproveEvent(t *testing.T, app *fiber.App, eventId string, approverId string) {
	t.Helper()
	resp := eaUpdateApproval(t, app, eventId, eventdomain.EventStatusApproved, validTokenFor(t, approverId))
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao aprovar o evento %s, recebeu %d", eventId, resp.StatusCode)
	}
}

func eaAddCommunityMember(t *testing.T, communityId string, userId string) {
	t.Helper()
	if _, err := communityUserRepository.Create(&communitydomain.CommunityUserDomain{
		CommunityId: communityId,
		UserId:      userId,
	}); err != nil {
		t.Fatalf("failed to add community member: %v", err)
	}
}

func eaCreateCommunityEvent(t *testing.T, app *fiber.App, owner *userdomain.UserDomain, communityId string, title string) eventdto.RegisterEventDto {
	t.Helper()
	return registerEventViaApi(t, app, eventTestRequest{
		OwnerId:     owner.Id,
		CommunityId: strPtr(communityId),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       title,
		Description: "evento de teste de aprovação",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
}

func TestEventApprovalFlow(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "appr_community_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "appr_member@ajuda.dev", userdomain.UserRoleUser)
	third := createUserWithRole(t, "appr_third@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "appr_city")
	community := createEventCommunity(t, "Comunidade aprovação", communityOwner, address)

	eaAddCommunityMember(t, community.Id, member.Id)

	ownerEvent := eaCreateCommunityEvent(t, app, communityOwner, community.Id, "Evento do dono da comunidade")
	if ownerEvent.Status != eventdomain.EventStatusApproved {
		t.Errorf("esperava APPROVED para evento criado pelo dono da comunidade, recebeu '%s'", ownerEvent.Status)
	}

	memberEvent := eaCreateCommunityEvent(t, app, member, community.Id, "Evento do membro")
	if memberEvent.Status != eventdomain.EventStatusPending {
		t.Errorf("esperava PENDING para evento criado por membro, recebeu '%s'", memberEvent.Status)
	}

	thirdToken := validTokenFor(t, third.Id)
	memberToken := validTokenFor(t, member.Id)
	communityOwnerToken := validTokenFor(t, communityOwner.Id)

	resp := euRequest(t, app, http.MethodGet, "/v1/event/"+memberEvent.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 no detalhe de evento PENDING para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+memberEvent.Id, "", memberToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no detalhe de evento PENDING para o criador, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+memberEvent.Id, "", communityOwnerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no detalhe de evento PENDING para o dono da comunidade, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem para terceiro, recebeu %d", resp.StatusCode)
	}
	page := eaDecodePageableEvent(t, resp)
	for _, item := range page.Data {
		if item.Id == memberEvent.Id {
			t.Errorf("esperava o evento PENDING fora da listagem do terceiro, recebeu %+v", item)
		}
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event", "", memberToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na listagem para o criador, recebeu %d", resp.StatusCode)
	}
	page = eaDecodePageableEvent(t, resp)
	found := false
	for _, item := range page.Data {
		if item.Id == memberEvent.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("esperava o evento PENDING na listagem do criador, recebeu %+v", page.Data)
	}

	resp = euRequest(t, app, http.MethodGet, "/v1/event?approval_status=PENDING&community_id="+community.Id, "", memberToken)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 no filtro approval_status para membro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event?approval_status=PENDING&community_id="+community.Id, "", communityOwnerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no filtro approval_status para o dono da comunidade, recebeu %d", resp.StatusCode)
	}
	page = eaDecodePageableEvent(t, resp)
	if len(page.Data) != 1 || page.Data[0].Id != memberEvent.Id {
		t.Errorf("esperava somente o evento PENDING na fila de aprovação, recebeu %+v", page.Data)
	}

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusApproved, memberToken)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 quando o criador tenta aprovar o próprio evento, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusApproved, thirdToken)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("esperava 403 quando um terceiro tenta aprovar, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusRejected, communityOwnerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao rejeitar o evento, recebeu %d", resp.StatusCode)
	}
	eventDto := eaDecodeEventDto(t, resp)
	if eventDto.Status != eventdomain.EventStatusRejected {
		t.Errorf("esperava status REJECTED na resposta, recebeu '%s'", eventDto.Status)
	}

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusRejected, communityOwnerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na transição REJECTED para REJECTED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusApproved, communityOwnerToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 na transição REJECTED para APPROVED, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusPending, communityOwnerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 na transição APPROVED para PENDING, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+memberEvent.Id, "", thirdToken)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no detalhe do evento aprovado para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, "invalid-id", eventdomain.EventStatusApproved, communityOwnerToken)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 para uuid inválido, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, uuidv7.New().String(), eventdomain.EventStatusApproved, communityOwnerToken)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 para evento inexistente, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, memberEvent.Id, eventdomain.EventStatusApproved, "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestEventApprovalNonMemberForbidden(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "appr_nm_owner@ajuda.dev", userdomain.UserRoleUser)
	outsider := createUserWithRole(t, "appr_nm_outsider@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "appr_nm_city")
	community := createEventCommunity(t, "Comunidade não membro", communityOwner, address)

	payload, err := json.Marshal(eventTestRequest{
		OwnerId:     outsider.Id,
		CommunityId: strPtr(community.Id),
		Category:    eventdomain.CategoryCommunityEvent,
		Type:        eventdomain.TypeOnline,
		Title:       "Evento de não membro",
		Description: "evento de teste de aprovação",
		StartAt:     time.Now().Add(48 * time.Hour),
		DurationMin: 60,
	})
	if err != nil {
		t.Fatalf("erro ao montar body: %v", err)
	}
	resp, err := doAuthedRequest(app, newEventRegisterRequest(payload), validTokenFor(t, outsider.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403 para não membro criando evento da comunidade, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "only community members can create events for this community" {
		t.Errorf("esperava a mensagem de 403 de não membro, recebeu '%s'", respBody.Message)
	}
}

func TestEventApprovalGatesParticipation(t *testing.T) {
	t.Cleanup(cleanAuthorizationData)

	app := setupApp()
	communityOwner := createUserWithRole(t, "appr_gate_owner@ajuda.dev", userdomain.UserRoleUser)
	member := createUserWithRole(t, "appr_gate_member@ajuda.dev", userdomain.UserRoleUser)
	attendee := createUserWithRole(t, "appr_gate_attendee@ajuda.dev", userdomain.UserRoleUser)
	speaker := createUserWithRole(t, "appr_gate_speaker@ajuda.dev", userdomain.UserRoleUser)
	address := createEventAddress(t, "appr_gate_city")
	community := createEventCommunity(t, "Comunidade gate", communityOwner, address)

	eaAddCommunityMember(t, community.Id, member.Id)

	pendingEvent := eaCreateCommunityEvent(t, app, member, community.Id, "Evento pendente do membro")
	if pendingEvent.Status != eventdomain.EventStatusPending {
		t.Fatalf("esperava PENDING para o evento do membro, recebeu '%s'", pendingEvent.Status)
	}

	resp := euJoinEvent(t, app, pendingEvent.Id, "", validTokenFor(t, attendee.Id))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 no join de evento PENDING, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodPost, "/v1/event/"+pendingEvent.Id+"/participants",
		`{"user_id":"`+speaker.Id+`","role":"SPEAKER"}`, validTokenFor(t, member.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 ao adicionar participante em evento PENDING pelo dono, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+pendingEvent.Id+"/participants", "", validTokenFor(t, attendee.Id))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 na listagem de participantes de evento PENDING para terceiro, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euRequest(t, app, http.MethodGet, "/v1/event/"+pendingEvent.Id+"/participants", "", validTokenFor(t, member.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 na listagem de participantes de evento PENDING para quem gerencia, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = eaUpdateApproval(t, app, pendingEvent.Id, eventdomain.EventStatusApproved, validTokenFor(t, communityOwner.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 ao aprovar o evento, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = euJoinEvent(t, app, pendingEvent.Id, "", validTokenFor(t, attendee.Id))
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 no join de evento aprovado, recebeu %d", resp.StatusCode)
	}
	resp.Body.Close()

	if status := eaEventStatus(t, pendingEvent.Id); status != eventdomain.EventStatusApproved {
		t.Errorf("esperava o evento persistido como APPROVED, recebeu '%s'", status)
	}
}
