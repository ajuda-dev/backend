package controller

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

func NewAuthController(authService service.AuthService) AuthController {
	return &authController{
		authService: authService,
	}
}

type AuthController interface {
	LoginUser() fiber.Handler
}

type authController struct {
	authService service.AuthService
}

// LoginUser godoc
// @Summary      Autentica um usuário
// @Description  Realiza o login e grava o token JWT em um cookie de sessão HttpOnly (ajudadev_session), enviado automaticamente pelo navegador nas próximas requisições. Clientes nativos informam o header X-Client-Type: native e recebem o token também no corpo da resposta, para guardar em secure storage e usar via Authorization: Bearer.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user           body    dto.LoginUserDtoIn  true   "Credenciais do usuário"
// @Param        X-Client-Type  header  string              false  "Informe 'native' (apps mobile/desktop) para receber o token JWT no corpo da resposta"
// @Success      200   {object}  dto.LoginUserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /v1/user/login [post]
func (a *authController) LoginUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var loginUserDto dto.LoginUserDtoIn
		if err := c.BodyParser(&loginUserDto); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		user, token, err := a.authService.LoginUser(loginUserDto.Email, loginUserDto.Password)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		middleware.SetSessionCookie(c, token)
		var loginUserDtoOut dto.LoginUserDtoOut
		loginUserDtoOut = *loginUserDtoOut.FromDomainUser(user)
		if middleware.WantsTokenInBody(c) {
			loginUserDtoOut.Token = token
		}
		return c.Status(fiber.StatusOK).JSON(loginUserDtoOut)
	}
}
