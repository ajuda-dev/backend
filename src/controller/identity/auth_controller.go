package identity

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service/identity"
	"github.com/gofiber/fiber/v2"
)

func NewAuthController(authService identity.AuthService) AuthController {
	return &authController{
		authService: authService,
	}
}

type AuthController interface {
	LoginUser() fiber.Handler
	ForgotPassword() fiber.Handler
	ResetPassword() fiber.Handler
}

type authController struct {
	authService identity.AuthService
}

// LoginUser godoc
// @Summary      Autentica um usuário
// @Description  Realiza o login e grava o token JWT em um cookie de sessão HttpOnly (ajudadev_session), enviado automaticamente pelo navegador nas próximas requisições. Clientes nativos informam o header X-Client-Type: native e recebem o token também no corpo da resposta, para guardar em secure storage e usar via Authorization: Bearer.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user           body    userdto.LoginUserDtoIn  true   "Credenciais do usuário"
// @Param        X-Client-Type  header  string              false  "Informe 'native' (apps mobile/desktop) para receber o token JWT no corpo da resposta"
// @Success      200   {object}  userdto.LoginUserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /v1/user/login [post]
func (a *authController) LoginUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var loginUserDto userdto.LoginUserDtoIn
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
		var loginUserDtoOut userdto.LoginUserDtoOut
		loginUserDtoOut = *loginUserDtoOut.FromDomainUser(user)
		if middleware.WantsTokenInBody(c) {
			loginUserDtoOut.Token = token
		}
		return c.Status(fiber.StatusOK).JSON(loginUserDtoOut)
	}
}

// ForgotPassword godoc
// @Summary      Solicita recuperação de senha
// @Description  Sempre responde 204. Se o e-mail existir e tiver senha, envia um código de 6 caracteres por e-mail. Não revela se a conta existe. Rate limit por e-mail (429).
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  userdto.ForgotPasswordDtoIn  true  "E-mail da conta"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      429   {object}  map[string]interface{}
// @Router       /v1/user/forgot-password [post]
func (a *authController) ForgotPassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body userdto.ForgotPasswordDtoIn
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		if err := a.authService.ForgotPassword(body.Email); err != nil {
			return c.Status(err.Code).JSON(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ResetPassword godoc
// @Summary      Redefine a senha com o código recebido por e-mail
// @Description  Valida o código HMAC (sem persistir código) e grava o novo hash bcrypt. Código inválido/expirado ou conta OAuth sem senha → 401 genérico. Rate limit de tentativas → 429.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  userdto.ResetPasswordDtoIn  true  "E-mail, código e nova senha"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      429   {object}  map[string]interface{}
// @Router       /v1/user/reset-password [post]
func (a *authController) ResetPassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body userdto.ResetPasswordDtoIn
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		if err := a.authService.ResetPassword(body.Email, body.Code, body.NewPassword); err != nil {
			return c.Status(err.Code).JSON(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}
