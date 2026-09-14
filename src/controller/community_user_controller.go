package controller

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type CommunityUserController interface {
	JoinCommunity() fiber.Handler
	LeaveCommunity() fiber.Handler
	GetCommunityMembers() fiber.Handler
}

type communityUserController struct {
	communityUserService service.CommunityUserService
}

func NewCommunityUserController(communityUserService service.CommunityUserService) CommunityUserController {
	return &communityUserController{
		communityUserService: communityUserService,
	}
}

// JoinCommunity godoc
// @Summary      Entra em uma comunidade
// @Description  Associa o usuário autenticado (do token) à comunidade como membro. A linha é única por (comunidade, usuário): tentar entrar de novo gera erro. O owner não possui linha própria de membership.
// @Tags         community_users
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID da comunidade"
// @Success      201   {object}  dto.CommunityUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id}/join [post]
func (c *communityUserController) JoinCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		communityId := cf.Params("id")
		if !uuidv7.IsValidString(communityId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		userId := cf.Locals(middleware.UserIdKey).(string)
		communityUser, err := c.communityUserService.JoinCommunity(communityId, userId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(dto.CommunityUserDto{}.FromDomain(communityUser))
	}
}

// LeaveCommunity godoc
// @Summary      Sai de uma comunidade
// @Description  Remove fisicamente a linha de membership do usuário autenticado na comunidade. O owner não é membro da própria comunidade: tentar sair dela retorna 404.
// @Tags         community_users
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID da comunidade"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id}/leave [delete]
func (c *communityUserController) LeaveCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		communityId := cf.Params("id")
		if !uuidv7.IsValidString(communityId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		userId := cf.Locals(middleware.UserIdKey).(string)
		if err := c.communityUserService.LeaveCommunity(communityId, userId); err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}

// GetCommunityMembers godoc
// @Summary      Lista membros da comunidade
// @Description  Retorna os membros de uma comunidade (linhas de community_users) com o usuário resumido, paginados em ordem alfabética pelo nome. O dono não aparece (não possui membership). Usuários sem e-mail compartilhado não expõem e-mail.
// @Tags         community_users
// @Accept       json
// @Produce      json
// @Param        id     path   string  true   "ID da comunidade"
// @Param        page   query  int     false  "Página"
// @Param        limit  query  int     false  "Limite"
// @Success      200   {object}  dto.PageableCommunityMemberDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/community/{id}/members [get]
func (c *communityUserController) GetCommunityMembers() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		communityId := cf.Params("id")
		if !uuidv7.IsValidString(communityId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		page, _ := strconv.Atoi(cf.Query("page", "1"))
		limit, _ := strconv.Atoi(cf.Query("limit", "10"))
		userId, ok := cf.Locals(middleware.UserIdKey).(string)
		if !ok || userId == "" {
			return cf.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing authenticated user"))
		}
		result, err := c.communityUserService.GetCommunityMembers(communityId, userId, page, limit)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.PageableCommunityMemberDto{}.FromDomain(*result))
	}
}
