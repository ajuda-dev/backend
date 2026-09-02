package controller

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller/dto"
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
