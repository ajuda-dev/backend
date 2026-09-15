package controller

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type CommunityController interface {
	RegisterCommunity() fiber.Handler
	GetCommunityById() fiber.Handler
	GetAllCommunities() fiber.Handler
	UpdateCommunity() fiber.Handler
	DeleteCommunity() fiber.Handler
}

type communityController struct {
	communityService service.CommunityService
}

func NewCommunityController(communityService service.CommunityService) CommunityController {
	return &communityController{
		communityService: communityService,
	}
}

// RegisterCommunity godoc
// @Summary      Registra uma nova comunidade
// @Description  Cria uma nova comunidade no sistema. O owner é sempre o usuário autenticado (owner_id do body é ignorado)
// @Tags         communities
// @Accept       json
// @Produce      json
// @Param        community  body  dto.RegisterCommunityDto  true  "Dados da comunidade"
// @Success      201   {object}  dto.CommunityDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/register [post]
func (c *communityController) RegisterCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		var registerCommunityDto dto.RegisterCommunityDto
		if err := cf.BodyParser(&registerCommunityDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}

		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		community := registerCommunityDto.ToDomain()
		community.Owner = domain.UserDomain{Id: userId}

		address, err := c.communityService.CreateCommunity(community)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(dto.CommunityDto{}.FromDomain(address))
	}
}

// GetCommunityById godoc
// @Summary      Busca comunidade por id
// @Description  Retorna o detalhe da comunidade com owner e endereço (preloads)
// @Tags         communities
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID da comunidade"
// @Success      200   {object}  dto.CommunityDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id} [get]
func (c *communityController) GetCommunityById() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		community, err := c.communityService.GetCommunityById(id)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.CommunityDto{}.FromDomain(community))
	}
}

// GetAllCommunities godoc
// @Summary      Lista comunidades
// @Description  Retorna todas as comunidades, com paginação e filtros por dono, nome e cidade (buscas parciais, case-insensitive e accent-insensitive)
// @Tags         communities
// @Accept       json
// @Produce      json
// @Param        page      query   int     false  "Página"
// @Param        limit     query   int     false  "Limite"
// @Param        owner_id  query   string  false  "ID do dono da comunidade (uuid v7)"
// @Param        name      query   string  false  "Nome da comunidade (busca parcial, case-insensitive, ignora acentos)"
// @Param        city      query   string  false  "Cidade do endereço da comunidade (busca parcial, case-insensitive, ignora acentos)"
// @Success      200   {object}  dto.PageableCommunityDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community [get]
func (c *communityController) GetAllCommunities() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		page, _ := strconv.Atoi(cf.Query("page", "1"))
		limit, _ := strconv.Atoi(cf.Query("limit", "10"))

		filter := repository.CommunityFilter{
			OwnerId: cf.Query("owner_id"),
			Name:    cf.Query("name"),
			City:    cf.Query("city"),
		}
		if filter.OwnerId != "" && !uuidv7.IsValidString(filter.OwnerId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid query params",
				[]rest_err.Causes{{Field: "owner_id", Message: "owner_id must be a valid UUID v7"}}))
		}
		result, e := c.communityService.GetAll(filter, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return cf.Status(e.Code).JSON(e)
		}

		dtoResult := dto.PageableCommunityDto{}.FromDomain(*result)
		return cf.Status(fiber.StatusOK).JSON(dtoResult)
	}
}

// UpdateCommunity godoc
// @Summary      Altera uma comunidade
// @Description  Atualiza nome, descrição e/ou endereço (campos vazios são ignorados). Somente o dono da comunidade, moderadores e admins.
// @Tags         communities
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "ID da comunidade"
// @Param        body  body  dto.UpdateCommunityDto  true  "Campos a alterar"
// @Success      200   {object}  dto.CommunityDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id} [put]
func (c *communityController) UpdateCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		var updateCommunityDto dto.UpdateCommunityDto
		if err := cf.BodyParser(&updateCommunityDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		requesterId := cf.Locals(middleware.UserIdKey).(string)
		community, err := c.communityService.UpdateCommunity(id, requesterId, updateCommunityDto.ToDomain())
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.CommunityDto{}.FromDomain(community))
	}
}

// DeleteCommunity godoc
// @Summary      Remove comunidade (soft delete)
// @Description  Arquiva a comunidade marcando deleted_at. Somente o owner pode deletar a própria comunidade (usuários comuns); moderadores e admins podem deletar qualquer comunidade. Comunidade com membros ativos não pode ser deletada.
// @Tags         communities
// @Param        id  path  string  true  "ID da comunidade"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id} [delete]
func (c *communityController) DeleteCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		requesterId := cf.Locals(middleware.UserIdKey).(string)
		if err := c.communityService.DeleteCommunity(id, requesterId); err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}
