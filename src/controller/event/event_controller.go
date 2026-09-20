package event

import (
	"strconv"
	"strings"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	eventdto "github.com/ajuda-dev/backend/src/controller/event/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	"github.com/ajuda-dev/backend/src/service/event"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type EventController interface {
	RegisterEvent() fiber.Handler
	GetEventById() fiber.Handler
	GetAllEvents() fiber.Handler
	RescheduleEvent() fiber.Handler
	DeleteEventById() fiber.Handler
	UpdateEventApproval() fiber.Handler
	UpdateEventVisibility() fiber.Handler
	UpdateEventMeetingLink() fiber.Handler
}

type eventController struct {
	eventService event.EventService
}

func NewEventController(eventService event.EventService) EventController {
	return &eventController{
		eventService: eventService,
	}
}

// RegisterEvent godoc
// @Summary      Registra um novo evento
// @Description  Cria um novo evento no sistema. Para eventos INPERSON/HYBRID, address_id é obrigatório; para ONLINE, address_id deve ser nulo. O owner é sempre o usuário autenticado (owner_id do body é ignorado). creator_role (MENTOR default ou MENTEE) só é aceito em eventos MENTORING e define o papel do criador no 1:1. MENTORING nasce CLOSED (não entra no catálogo); WEBINAR nasce PUBLIC; COMMUNITY_EVENT nasce CLOSED. meeting_link é opcional (http/https com host, máx. 500; só ONLINE/HYBRID).
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        event  body  eventdto.RegisterEventDto  true  "Dados do evento"
// @Success      201   {object}  eventdto.RegisterEventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/register [post]
func (e *eventController) RegisterEvent() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		var registerEventDto eventdto.RegisterEventDto
		if err := cf.BodyParser(&registerEventDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}

		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		event := registerEventDto.ToDomain()
		event.Owner = userdomain.UserDomain{Id: userId}

		eventResult, err := e.eventService.CreateEvent(event)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(registerEventDto.FromDomain(eventResult))
	}
}

// GetEventById godoc
// @Summary      Busca evento por id
// @Description  Retorna o detalhe do evento com owner, endereço e comunidade (preloads). MENTORING CLOSED e COMMUNITY_EVENT CLOSED: terceiro recebe 404. meeting_link só para owner, participante ativo (status ≠ CANCELLED) ou staff; omitido no JSON para os demais.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Success      200   {object}  eventdto.EventDto
// @Failure      404   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
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
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		event, err := e.eventService.GetEventDetail(id, userId)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventDto{}.FromDomain(event))
	}
}

// GetAllEvents godoc
// @Summary      Lista eventos
// @Description  Catálogo paginado: só eventos APPROVED com visibility PUBLIC (ou vazia), mais os do requester (owner, dono da comunidade ou participante ativo). MENTORING CLOSED não aparece para terceiro; use ?user_id= self para a agenda. meeting_link omitido para quem não participa. Staff lista também não aprovados/fechados.
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
// @Param        approval_status query  string  false  "Status de aprovação do evento (PENDING, APPROVED, REJECTED) — apenas dono da comunidade ou staff"
// @Success      200   {object}  eventdto.PageableEventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event [get]
func (e *eventController) GetAllEvents() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		page, _ := strconv.Atoi(cf.Query("page", "1"))
		limit, _ := strconv.Atoi(cf.Query("limit", "10"))

		filter := eventrepo.EventFilter{
			CommunityId:    cf.Query("community_id"),
			Category:       cf.Query("category"),
			Type:           cf.Query("type"),
			AddressId:      cf.Query("address_id"),
			City:           cf.Query("city"),
			Upcoming:       strings.EqualFold(cf.Query("upcoming"), "true"),
			UserId:         cf.Query("user_id"),
			Role:           cf.Query("role"),
			Status:         cf.Query("status"),
			ApprovalStatus: cf.Query("approval_status"),
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

		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}

		result, e := e.eventService.GetAll(filter, page, limit, userId)
		if e != nil {
			logger.Error("error: ", e)
			return cf.Status(e.Code).JSON(e)
		}

		dtoResult := eventdto.PageableEventDto{}.FromDomain(*result)
		return cf.Status(fiber.StatusOK).JSON(dtoResult)
	}
}

// RescheduleEvent godoc
// @Summary      Reagenda um evento
// @Description  Quem gerencia o evento (ou o convidado: MENTORING, ou SPEAKER em COMMUNITY_EVENT/WEBINAR) altera start_at e grava um comment obrigatório (trim, máx. 500) na linha event_users de quem reagenda (comment_kind=RESCHEDULE). Reagendar de novo sobrescreve o comment anterior do ator. Não muda a aprovação. Quem reagenda fica CONFIRMED; em MENTORING o outro CONFIRMED volta para REQUESTED; em COMMUNITY_EVENT/WEBINAR os outros palestrantes CONFIRMED voltam para REQUESTED e inscritos ATTENDEE permanecem CONFIRMED.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Param        body  body  eventdto.RescheduleEventDto  true  "Nova data (futuro) e comment obrigatório"
// @Success      200   {object}  eventdto.EventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id}/reschedule [put]
func (e *eventController) RescheduleEvent() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		var rescheduleDto eventdto.RescheduleEventDto
		if err := cf.BodyParser(&rescheduleDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		event, err := e.eventService.Reschedule(id, userId, rescheduleDto.StartAt, rescheduleDto.Comment)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventDto{}.FromDomain(event))
	}
}

// DeleteEventById godoc
// @Summary      Remove evento (soft delete)
// @Description  Arquiva o evento marcando deleted_at. Body exige comment (trim, máx. 500); o texto fica em event_users.status_comment do ator (comment_kind=CANCEL) na mesma transação do delete.
// @Tags         events
// @Accept       json
// @Param        id  path  string  true  "ID do evento"
// @Param        body  body  eventdto.CancelEventDto  true  "comment obrigatório do cancelamento"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
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
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		var cancelDto eventdto.CancelEventDto
		if err := cf.BodyParser(&cancelDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		err := e.eventService.DeleteEventById(id, userId, cancelDto.Comment)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}

// UpdateEventApproval godoc
// @Summary      Aprova ou rejeita um evento da comunidade
// @Description  O dono da comunidade (ou moderador/admin) decide o status de aprovação do evento. Transições permitidas: PENDING para APPROVED ou REJECTED e REJECTED para APPROVED.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Param        body  body  eventdto.UpdateEventApprovalDto  true  "Novo status (APPROVED ou REJECTED)"
// @Success      200   {object}  eventdto.EventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id}/approval [put]
func (e *eventController) UpdateEventApproval() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		var updateEventApprovalDto eventdto.UpdateEventApprovalDto
		if err := cf.BodyParser(&updateEventApprovalDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventResult, err := e.eventService.UpdateApproval(id, userId, updateEventApprovalDto.Status)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventDto{}.FromDomain(eventResult))
	}
}

// UpdateEventVisibility godoc
// @Summary      Publica um evento da comunidade
// @Description  Quem gerencia o evento (dono, dono da comunidade ou staff) torna um COMMUNITY_EVENT CLOSED em PUBLIC. Exige palestrante CONFIRMED e que o owner não esteja HOST REQUESTED. Se o ator puder aprovar e o evento estiver PENDING, a publicação também aprova.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Param        body  body  eventdto.UpdateEventVisibilityDto  true  "visibility PUBLIC"
// @Success      200   {object}  eventdto.EventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id}/visibility [put]
func (e *eventController) UpdateEventVisibility() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		var updateEventVisibilityDto eventdto.UpdateEventVisibilityDto
		if err := cf.BodyParser(&updateEventVisibilityDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventResult, err := e.eventService.UpdateVisibility(id, userId, updateEventVisibilityDto.Visibility)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventDto{}.FromDomain(eventResult))
	}
}

// UpdateEventMeetingLink godoc
// @Summary      Atualiza o link da reunião
// @Description  Quem gerencia o evento (dono, dono da comunidade ou staff) inclui, altera ou limpa meeting_link. Vazio limpa o campo. Só ONLINE e HYBRID; URL http/https com host, máx. 500.
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID do evento"
// @Param        body  body  eventdto.UpdateEventMeetingLinkDto  true  "meeting_link (vazio limpa)"
// @Success      200   {object}  eventdto.EventDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/event/{id}/meeting-link [put]
func (e *eventController) UpdateEventMeetingLink() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		var updateEventMeetingLinkDto eventdto.UpdateEventMeetingLinkDto
		if err := cf.BodyParser(&updateEventMeetingLinkDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		eventResult, err := e.eventService.UpdateMeetingLink(id, userId, updateEventMeetingLinkDto.MeetingLink)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(eventdto.EventDto{}.FromDomain(eventResult))
	}
}
