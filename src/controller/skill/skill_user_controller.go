package controller

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type SkillUserController interface {
	AssignSkill() fiber.Handler
	GetUserSkills() fiber.Handler
	RemoveSkillFromUser() fiber.Handler
}

type skillUserController struct {
	skillUserService service.SkillUserService
}

func NewSkillUserController(skillUserService service.SkillUserService) SkillUserController {
	return &skillUserController{
		skillUserService: skillUserService,
	}
}

// AssignSkill godoc
// @Summary      Adiciona skill ao perfil do usuário
// @Description  Associa uma skill do catálogo a um usuário com um nível (WANT_TO_LEARN, LEARN_AND_TEACH ou TEACH). A linha é única por (skill, usuário): tentar associar de novo gera erro.
// @Tags         skill_users
// @Accept       json
// @Produce      json
// @Param        skillId  path  string  true  "ID da skill"
// @Param        body  body  dto.AssignSkillDto  true  "Dados da associação"
// @Success      201   {object}  dto.SkillUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill/{skillId}/users [post]
func (s *skillUserController) AssignSkill() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		skillId := cf.Params("skillId")
		if !uuidv7.IsValidString(skillId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "skillId", Message: "skillId must be a valid UUID v7"}}))
		}
		var assignSkillDto dto.AssignSkillDto
		if err := cf.BodyParser(&assignSkillDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		skillUser, err := s.skillUserService.AssignSkill(assignSkillDto.ToDomain(skillId))
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(dto.SkillUserDto{}.FromDomain(skillUser))
	}
}

// GetUserSkills godoc
// @Summary      Lista skills do perfil do usuário
// @Description  Retorna as skills ativas do usuário (com nível), em ordem alfabética. Usuário válido sem skills retorna lista vazia.
// @Tags         skill_users
// @Accept       json
// @Produce      json
// @Param        userId  path  string  true  "ID do usuário"
// @Success      200   {array}   dto.SkillUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/{userId}/skills [get]
func (s *skillUserController) GetUserSkills() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		userId := cf.Params("userId")
		if !uuidv7.IsValidString(userId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		skills, err := s.skillUserService.GetUserSkills(userId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.ToSkillUserDtoList(skills))
	}
}

// RemoveSkillFromUser godoc
// @Summary      Remove skill do perfil do usuário
// @Description  Remove fisicamente a associação entre o usuário e a skill
// @Tags         skill_users
// @Accept       json
// @Produce      json
// @Param        userId   path  string  true  "ID do usuário"
// @Param        skillId  path  string  true  "ID da skill"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/{userId}/skills/{skillId} [delete]
func (s *skillUserController) RemoveSkillFromUser() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		userId := cf.Params("userId")
		skillId := cf.Params("skillId")
		if !uuidv7.IsValidString(userId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		if !uuidv7.IsValidString(skillId) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "skillId", Message: "skillId must be a valid UUID v7"}}))
		}
		err := s.skillUserService.RemoveSkillFromUser(userId, skillId)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}
