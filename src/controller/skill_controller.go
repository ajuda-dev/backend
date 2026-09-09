package controller

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

type SkillController interface {
	RegisterSkill() fiber.Handler
	GetSkillById() fiber.Handler
	GetAllSkills() fiber.Handler
	UpdateSkill() fiber.Handler
	DeleteSkill() fiber.Handler
}

type skillController struct {
	skillService service.SkillService
}

func NewSkillController(skillService service.SkillService) SkillController {
	return &skillController{
		skillService: skillService,
	}
}

// RegisterSkill godoc
// @Summary      Registra uma nova skill
// @Description  Cria uma skill no catálogo. O nome é salvo em caixa alta (normalizado) e é único entre skills ativas.
// @Tags         skills
// @Accept       json
// @Produce      json
// @Param        skill  body  dto.RegisterSkillDto  true  "Dados da skill"
// @Success      201   {object}  dto.RegisterSkillDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill/register [post]
func (s *skillController) RegisterSkill() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		var registerSkillDto dto.RegisterSkillDto
		if err := cf.BodyParser(&registerSkillDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}

		skill, err := s.skillService.CreateSkill(registerSkillDto.ToDomain())
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(registerSkillDto.FromDomain(skill))
	}
}

// GetSkillById godoc
// @Summary      Busca skill por id
// @Description  Retorna o detalhe da skill
// @Tags         skills
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "ID da skill"
// @Success      200   {object}  dto.SkillDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill/{id} [get]
func (s *skillController) GetSkillById() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		skill, err := s.skillService.GetSkillById(id)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.SkillDto{}.FromDomain(skill))
	}
}

// GetAllSkills godoc
// @Summary      Lista skills do catálogo
// @Description  Retorna as skills com paginação e busca por prefixo do nome (case insensitive: o termo é normalizado para caixa alta antes do LIKE)
// @Tags         skills
// @Accept       json
// @Produce      json
// @Param        name   query  string  false  "Prefixo do nome da skill"
// @Param        page   query  int     false  "Página"
// @Param        limit  query  int     false  "Limite"
// @Success      200   {object}  dto.PageableSkillDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill [get]
func (s *skillController) GetAllSkills() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		page, _ := strconv.Atoi(cf.Query("page", "1"))
		limit, _ := strconv.Atoi(cf.Query("limit", "10"))

		result, e := s.skillService.GetAll(repository.SkillFilter{
			Name: cf.Query("name"),
		}, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return cf.Status(e.Code).JSON(e)
		}

		dtoResult := dto.PageableSkillDto{}.FromDomain(*result)
		return cf.Status(fiber.StatusOK).JSON(dtoResult)
	}
}

// UpdateSkill godoc
// @Summary      Renomeia uma skill
// @Description  Atualiza o nome da skill (normalizado para caixa alta). Nome já usado por outra skill ativa gera conflito.
// @Tags         skills
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "ID da skill"
// @Param        body  body  dto.UpdateSkillDto  true  "Novo nome da skill"
// @Success      200   {object}  dto.RegisterSkillDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill/{id} [put]
func (s *skillController) UpdateSkill() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		var updateSkillDto dto.UpdateSkillDto
		if err := cf.BodyParser(&updateSkillDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		skill, err := s.skillService.UpdateSkill(id, updateSkillDto.Name)
		if err != nil {
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusOK).JSON(dto.RegisterSkillDto{}.FromDomain(skill))
	}
}

// DeleteSkill godoc
// @Summary      Remove skill (soft delete)
// @Description  Arquiva a skill marcando deleted_at e remove as associações skill_users dela (a skill some do perfil de todos os usuários)
// @Tags         skills
// @Param        id  path  string  true  "ID da skill"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/skill/{id} [delete]
func (s *skillController) DeleteSkill() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		id := cf.Params("id")
		if !uuidv7.IsValidString(id) {
			return cf.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a valid UUID v7"}}))
		}
		err := s.skillService.DeleteSkill(id)
		if err != nil {
			logger.Error("error: ", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.SendStatus(fiber.StatusNoContent)
	}
}
