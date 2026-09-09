package controller

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

func NewUserController(userService service.UserService) UserController {
	return &userController{
		userService: userService,
	}
}

type UserController interface {
	RegisterUser() fiber.Handler
	GetAllUsers() fiber.Handler
}

type userController struct {
	userService service.UserService
}

// RegisterUser godoc
// @Summary      Registra um novo usuário
// @Description  Cria um novo usuário no sistema
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body  dto.RegisterUserDtoIn  true  "Dados do usuário"
// @Success      201   {object}  dto.UserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Router       /v1/user/register [post]
func (u *userController) RegisterUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var registerUserDto dto.RegisterUserDtoIn
		if err := c.BodyParser(&registerUserDto); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		user, token, err := u.userService.CreateUser(registerUserDto.ToDomain())
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		var registerUserDtoOut dto.UserDtoOut
		registerUserDtoOut = *registerUserDtoOut.FromDomainUser(user)
		registerUserDtoOut.Token = token
		return c.Status(fiber.StatusCreated).JSON(registerUserDtoOut)
	}
}

// GetAllUsers godoc
// @Summary      Busca usuários por skill
// @Description  Retorna os usuários ativos que possuem a skill informada (match exato do nome, normalizado para caixa alta) com as skills do perfil de cada um. O parâmetro skill é obrigatório.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        skill  query  string  true  "Nome da skill (normalizado para caixa alta)"
// @Param        page   query  int     false  "Página"
// @Param        limit  query  int     false  "Limite"
// @Success      200   {object}  dto.PageableUserDto
// @Failure      400   {object}  map[string]interface{}
// @Router       /v1/user [get]
func (u *userController) GetAllUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "10"))

		result, e := u.userService.GetAllUsers(repository.UserFilter{
			SkillName: c.Query("skill"),
		}, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return c.Status(e.Code).JSON(e)
		}

		dtoResult := dto.PageableUserDto{}.FromDomain(*result)
		return c.Status(fiber.StatusOK).JSON(dtoResult)
	}
}
