package controller

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service"
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
	eventUserService service.EventUserService
}

func NewEventUserController(eventUserService service.EventUserService) EventUserController {
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
// @Param        body  body  dto.JoinEventDto  true  "Dados da inscrição"
// @Success      201   {object}  dto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
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
		var joinEventDto dto.JoinEventDto
		if err := cf.BodyParser(&joinEventDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		eventUser, err := e.eventUserService.JoinEvent(joinEventDto.ToDomain(eventId))
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(dto.EventUserDto{}.FromDomain(eventUser))
	}
}

// AddParticipant godoc
// @Summary      Adiciona participante (MENTEE ou SPEAKER)
// @Description  O host adiciona um MENTEE (status REQUESTED) em eventos MENTORING ou um SPEAKER (status CONFIRMED) em eventos COMMUNITY_EVENT/WEBINAR.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        body  body  dto.AddParticipantDto  true  "Dados do participante"
// @Success      201   {object}  dto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
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
		var addParticipantDto dto.AddParticipantDto
		if err := cf.BodyParser(&addParticipantDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		eventUser, err := e.eventUserService.AddParticipant(addParticipantDto.ToDomain(eventId))
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(dto.EventUserDto{}.FromDomain(eventUser))
	}
}

// GetParticipants godoc
// @Summary      Lista participantes do evento
// @Description  Retorna os participantes do evento com usuário (preload), com filtro opcional de status
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        status   query  string  false  "Filtro por status (REQUESTED, CONFIRMED, REJECTED, CANCELLED)"
// @Success      200   {array}   dto.EventUserDto
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
		participants, err := e.eventUserService.GetParticipants(eventId, cf.Query("status"))
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.ToEventUserDtoList(participants))
	}
}

// UpdateParticipantStatus godoc
// @Summary      Atualiza status do participante
// @Description  MENTEE aceita (CONFIRMED) ou recusa (REJECTED) o convite de mentoria. Garante no máximo 1 MENTEE CONFIRMED por evento MENTORING.
// @Tags         event_users
// @Accept       json
// @Produce      json
// @Param        eventId  path  string  true  "ID do evento"
// @Param        userId   path  string  true  "ID do usuário"
// @Param        body  body  dto.UpdateParticipantStatusDto  true  "Novo status (CONFIRMED ou REJECTED)"
// @Success      200   {object}  dto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
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
		var updateStatusDto dto.UpdateParticipantStatusDto
		if err := cf.BodyParser(&updateStatusDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		eventUser, err := e.eventUserService.UpdateParticipantStatus(updateStatusDto.ToDomain(eventId, userId))
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.EventUserDto{}.FromDomain(eventUser))
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
// @Success      200   {object}  dto.EventUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
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
		eventUser, err := e.eventUserService.CancelParticipation(eventId, userId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.EventUserDto{}.FromDomain(eventUser))
	}
}
