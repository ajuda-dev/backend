package controller

import (
	"strconv"
	"strings"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type EventController interface {
	RegisterEvent() fiber.Handler
	GetEventById() fiber.Handler
	GetAllEvents() fiber.Handler
	DeleteEventById() fiber.Handler
}

type eventController struct {
	eventService service.EventService
}

func NewEventController(eventService service.EventService) EventController {
	return &eventController{
		eventService: eventService,
	}
}

// RegisterEvent godoc
// @Summary      Registra um novo evento
// @Description  Cria um novo evento no sistema. Para eventos INPERSON/HYBRID, address_id é obrigatório; para ONLINE, address_id deve ser nulo.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        event  body  dto.RegisterEventDto  true  "Dados do evento"
// @Success      201   {object}  dto.RegisterEventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/register [post]
func (e *eventController) RegisterEvent() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		var registerEventDto dto.RegisterEventDto
		if err := cf.BodyParser(&registerEventDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}

		event, err := e.eventService.CreateEvent(registerEventDto.ToDomain())
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(registerEventDto.FromDomain(event))
	}
}

// GetEventById godoc
// @Summary      Busca evento por id
// @Description  Retorna o detalhe do evento com owner, endereço e comunidade (preloads)
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Success      200   {object}  dto.EventDto
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id} [get]
func (e *eventController) GetEventById() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		event, err := e.eventService.GetEventById(id)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.EventDto{}.FromDomain(event))
	}
}

// GetAllEvents godoc
// @Summary      Lista eventos
// @Description  Retorna todos os eventos, com paginação e filtros por comunidade, categoria, tipo, endereço, cidade e apenas futuros
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        page         query  int     false  "Página"
// @Param        limit        query  int     false  "Limite"
// @Param        community_id query  string  false  "ID da comunidade"
// @Param        category     query  string  false  "Categoria (COMMUNITY_EVENT, MENTORING, WEBINAR)"
// @Param        type         query  string  false  "Formato (ONLINE, INPERSON, HYBRID)"
// @Param        address_id   query  string  false  "ID do endereço"
// @Param        city         query  string  false  "Cidade do endereço do evento"
// @Param        upcoming     query  boolean false  "Somente eventos futuros"
// @Param        user_id      query  string  false  "ID do usuário (agenda: eventos com participação)"
// @Param        role         query  string  false  "Papel da participação (com user_id)"
// @Param        status       query  string  false  "Status da participação (com user_id)"
// @Success      200   {object}  dto.PageableEventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event [get]
func (e *eventController) GetAllEvents() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		page, _ := strconv.Atoi(cf.Query("page", "1"))
		limit, _ := strconv.Atoi(cf.Query("limit", "10"))

		filter := repository.EventFilter{
			CommunityId: cf.Query("community_id"),
			Category:    cf.Query("category"),
			Type:        cf.Query("type"),
			AddressId:   cf.Query("address_id"),
			City:        cf.Query("city"),
			Upcoming:    strings.EqualFold(cf.Query("upcoming"), "true"),
			UserId:      cf.Query("user_id"),
			Role:        cf.Query("role"),
			Status:      cf.Query("status"),
		}
		if filter.CommunityId != "" && !uuidv7.IsValidString(filter.CommunityId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid query params",
				[]rest_err.Causes{{Field: "community_id", Message: "community_id must be a valid UUID v7"}}))
		}
		if filter.AddressId != "" && !uuidv7.IsValidString(filter.AddressId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid query params",
				[]rest_err.Causes{{Field: "address_id", Message: "address_id must be a valid UUID v7"}}))
		}
		if filter.UserId != "" && !uuidv7.IsValidString(filter.UserId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid query params",
				[]rest_err.Causes{{Field: "user_id", Message: "user_id must be a valid UUID v7"}}))
		}

		result, e := e.eventService.GetAll(filter, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return cf.Status(e.Code).JSON(e)
		}

		dtoResult := dto.PageableEventDto{}.FromDomain(*result)
		return cf.Status(fiber.StatusOK).JSON(dtoResult)
	}
}

// DeleteEventById godoc
// @Summary      Remove evento (soft delete)
// @Description  Arquiva o evento marcando deleted_at
// @Tags         events
// @Param        id  path  string  true  "ID do evento"
// @Success      204
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id} [delete]
func (e *eventController) DeleteEventById() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		err := e.eventService.DeleteEventById(id)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}
