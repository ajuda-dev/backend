package event

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service/event"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type EventUserController interface {
	JoinEvent() fiber.Handler
	AddParticipant() fiber.Handler
	GetParticipants() fiber.Handler
	UpdateParticipantStatus() fiber.Handler
	CancelParticipation() fiber.Handler
}

type eventUserController struct {
	eventUserService event.EventUserService
}

func NewEventUserController(eventUserService event.EventUserService) EventUserController {
	return &eventUserController{
		eventUserService: eventUserService,
	}
}

// JoinEvent godoc
// @Summary      Auto-inscreve um participante
// @Description  Inscreve o usuário como ATTENDEE em um evento COMMUNITY_EVENT ou WEBINAR, respeitando MaxSlots. A linha é única por (evento, usuário): cancelamentos viram CANCELLED e uma nova inscrição reativa a linha.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Success      201   {object}  eventdto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{eventId}/join [post]
func (e *eventUserController) JoinEvent() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		eventId := cf.Params("eventId")
		if !uuidv7.IsValidString(eventId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "eventId", Message: "eventId must be a valid UUID v7"}}))
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventUser, err := e.eventUserService.JoinEvent(eventId, userId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(eventdto.EventUserDto{}.FromDomain(eventUser))
	}
}

// AddParticipant godoc
// @Summary      Adiciona participante (MENTOR/MENTEE ou SPEAKER)
// @Description  Quem gerencia o evento convida um participante. Em MENTORING o papel deve ser complementar ao creator_role de quem criou (MENTOR convida MENTEE e vice-versa) e a linha nasce REQUESTED; em COMMUNITY_EVENT/WEBINAR o convidado é SPEAKER e a linha nasce CONFIRMED.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        body  body  eventdto.AddParticipantDto  true  "Dados do participante"
// @Success      201   {object}  eventdto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{eventId}/participants [post]
func (e *eventUserController) AddParticipant() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		eventId := cf.Params("eventId")
		if !uuidv7.IsValidString(eventId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "eventId", Message: "eventId must be a valid UUID v7"}}))
		}
		var addParticipantDto eventdto.AddParticipantDto
		if err := cf.BodyParser(&addParticipantDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventUser, err := e.eventUserService.AddParticipant(eventId, userId, addParticipantDto.UserId, addParticipantDto.Role)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(eventdto.EventUserDto{}.FromDomain(eventUser))
	}
}

// GetParticipants godoc
// @Summary      Lista participantes do evento
// @Description  Retorna os participantes do evento com usuário (preload), com filtro opcional de status. Itens podem incluir comment após accept/reject.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        status   query  string  false  "Filtro por status (REQUESTED, CONFIRMED, REJECTED, CANCELLED)"
// @Success      200   {array}   eventdto.EventParticipantDto
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{eventId}/participants [get]
func (e *eventUserController) GetParticipants() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		eventId := cf.Params("eventId")
		if !uuidv7.IsValidString(eventId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "eventId", Message: "eventId must be a valid UUID v7"}}))
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		participants, err := e.eventUserService.GetParticipants(eventId, cf.Query("status"), userId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.ToEventParticipantDtoList(participants))
	}
}

// UpdateParticipantStatus godoc
// @Summary      Atualiza status do participante
// @Description  O convidado aceita (CONFIRMED, comment opcional) ou recusa (REJECTED, comment obrigatório após trim, máx. 500). O texto fica no evento (resposta e GET participants) e não entra na notificação. Garante no máximo 1 MENTEE CONFIRMED por evento MENTORING.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        userId   path  string  true  "ID do usuário"
// @Param        body  body  eventdto.UpdateParticipantStatusDto  true  "Status (CONFIRMED ou REJECTED) e comment (obrigatório em REJECTED)"
// @Success      200   {object}  eventdto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{eventId}/participants/{userId}/status [put]
func (e *eventUserController) UpdateParticipantStatus() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		eventId := cf.Params("eventId")
		userId := cf.Params("userId")
		if !uuidv7.IsValidString(eventId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "eventId", Message: "eventId must be a valid UUID v7"}}))
		}
		if !uuidv7.IsValidString(userId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		var updateStatusDto eventdto.UpdateParticipantStatusDto
		if err := cf.BodyParser(&updateStatusDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		requesterId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || requesterId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventUser, err := e.eventUserService.UpdateParticipantStatus(eventId, userId, requesterId, updateStatusDto.Status, updateStatusDto.Comment)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventUserDto{}.FromDomain(eventUser))
	}
}

// CancelParticipation godoc
// @Summary      Cancela participação (desistência)
// @Description  Não deleta a linha: o status vira CANCELLED. Reinscrever depois volta a mesma linha para CONFIRMED.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        userId   path  string  true  "ID do usuário"
// @Success      200   {object}  eventdto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{eventId}/participants/{userId} [delete]
func (e *eventUserController) CancelParticipation() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		eventId := cf.Params("eventId")
		userId := cf.Params("userId")
		if !uuidv7.IsValidString(eventId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "eventId", Message: "eventId must be a valid UUID v7"}}))
		}
		if !uuidv7.IsValidString(userId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		requesterId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || requesterId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventUser, err := e.eventUserService.CancelParticipation(eventId, userId, requesterId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventUserDto{}.FromDomain(eventUser))
	}
}
