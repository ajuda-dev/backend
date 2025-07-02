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

// RegisterUser implements UserController.
func (u *userController) RegisterUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var registerUserDto dto.RegisterUserDtoIn
		if err := c.BodyParser(&registerUserDto); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		user, err := u.userService.CreateUser(registerUserDto.ToDomain())
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		var registerUserDtoOut  dto.RegisterUserDtoOut
		return c.Status(fiber.StatusCreated).JSON(registerUserDtoOut.FromDomainUser(user))
	}
}
