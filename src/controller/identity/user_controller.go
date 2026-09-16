package identity

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	"github.com/ajuda-dev/backend/src/service/identity"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func NewUserController(userService identity.UserService) UserController {
	return &userController{
		userService: userService,
	}
}

type UserController interface {
	RegisterUser() fiber.Handler
	Logout() fiber.Handler
	Me() fiber.Handler
	VerifyEmail() fiber.Handler
	ResendVerification() fiber.Handler
	GetAllUsers() fiber.Handler
	GetUserById() fiber.Handler
	UpdateUser() fiber.Handler
	DeleteUser() fiber.Handler
}

type userController struct {
	userService identity.UserService
}

// RegisterUser godoc
// @Summary      Registra um novo usuário
// @Description  Cria um novo usuário com e-mail ainda não confirmado (emailVerified false) e grava o token JWT em um cookie de sessão HttpOnly (ajudadev_session), enviado automaticamente pelo navegador nas próximas requisições. O worker da outbox envia o código de confirmação. Clientes nativos informam o header X-Client-Type: native e recebem o token também no corpo da resposta, para guardar em secure storage e usar via Authorization: Bearer.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user           body    userdto.RegisterUserDtoIn  true   "Dados do usuário"
// @Param        X-Client-Type  header  string                 false  "Informe 'native' (apps mobile/desktop) para receber o token JWT no corpo da resposta"
// @Success      201   {object}  userdto.UserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Router       /v1/user/register [post]
func (u *userController) RegisterUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var registerUserDto userdto.RegisterUserDtoIn
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
		middleware.SetSessionCookie(c, token)
		var registerUserDtoOut userdto.UserDtoOut
		registerUserDtoOut = *registerUserDtoOut.FromDomainUser(user)
		if middleware.WantsTokenInBody(c) {
			registerUserDtoOut.Token = token
		}
		return c.Status(fiber.StatusCreated).JSON(registerUserDtoOut)
	}
}

// Me godoc
// @Summary      Retorna o usuário autenticado
// @Description  Devolve id, name, email, role e emailVerified do usuário da sessão atual (cookie de sessão ou Authorization: Bearer). Usado pelo frontend no boot para restaurar a sessão sem guardar o token no navegador.
// @Tags         users
// @Produce      json
// @Success      200   {object}  userdto.UserDtoOut
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/me [get]
func (u *userController) Me() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requesterId := c.Locals(middleware.UserIdKey).(string)
		user, err := u.userService.GetUserById(requesterId, requesterId)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		var userDtoOut userdto.UserDtoOut
		userDtoOut = *userDtoOut.FromDomainUser(user)
		return c.Status(fiber.StatusOK).JSON(userDtoOut)
	}
}

// VerifyEmail godoc
// @Summary      Confirma o e-mail com o código recebido
// @Description  Valida o código de 6 caracteres (0-9A-Z) enviado no cadastro. Sucesso preenche email_verified_at e devolve o usuário com emailVerified true. Código inválido/expirado → 401. Rate limit de tentativas → 429.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  userdto.VerifyEmailDtoIn  true  "Código de confirmação"
// @Success      200   {object}  userdto.UserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      429   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/verify-email [post]
func (u *userController) VerifyEmail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requesterId := c.Locals(middleware.UserIdKey).(string)
		var body userdto.VerifyEmailDtoIn
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		user, err := u.userService.VerifyEmail(requesterId, body.Code)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		var userDtoOut userdto.UserDtoOut
		userDtoOut = *userDtoOut.FromDomainUser(user)
		return c.Status(fiber.StatusOK).JSON(userDtoOut)
	}
}

// ResendVerification godoc
// @Summary      Reenvia o código de confirmação de e-mail
// @Description  Se o e-mail ainda não foi confirmado, enfileira um novo evento CREATED_ACCOUNT na outbox. Sempre 204 quando a conta já está verificada. Rate limit de envios → 429.
// @Tags         users
// @Success      204
// @Failure      401   {object}  map[string]interface{}
// @Failure      429   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/resend-verification [post]
func (u *userController) ResendVerification() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requesterId := c.Locals(middleware.UserIdKey).(string)
		if err := u.userService.ResendVerification(requesterId); err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// Logout godoc
// @Summary      Encerra a sessão (logout)
// @Description  Limpa o cookie HttpOnly de sessão. Endpoint público e idempotente: precisa responder mesmo com a sessão já expirada, para que o navegador pare de enviar o cookie.
// @Tags         users
// @Success      204
// @Router       /v1/user/logout [post]
func (u *userController) Logout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		middleware.ClearSessionCookie(c)
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// GetAllUsers godoc
// @Summary      Lista/busca usuários
// @Description  Retorna usuários ativos com as skills do perfil, em ordem alfabética por nome. Filtros opcionais e combináveis (AND): skill (match exato do nome, normalizado para caixa alta, máx. 50), name (busca parcial, case-insensitive, ignora acentos) e email (match exato, case-insensitive). Sem filtros, retorna todos os usuários ativos paginados. O e-mail não é exposto no response.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        skill  query  string  false  "Nome da skill (match exato, normalizado para caixa alta, máx. 50)"
// @Param        name   query  string  false  "Nome do usuário (busca parcial, case-insensitive, ignora acentos)"
// @Param        email  query  string  false  "E-mail do usuário (match exato, case-insensitive)"
// @Param        page   query  int     false  "Página"
// @Param        limit  query  int     false  "Limite"
// @Success      200   {object}  userdto.PageableUserDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user [get]
func (u *userController) GetAllUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "10"))

		result, e := u.userService.GetAllUsers(userrepo.UserFilter{
			SkillName: c.Query("skill"),
			Name:      c.Query("name"),
			Email:     c.Query("email"),
		}, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return c.Status(e.Code).JSON(e)
		}

		dtoResult := userdto.PageableUserDto{}.FromDomain(*result)
		return c.Status(fiber.StatusOK).JSON(dtoResult)
	}
}

// GetUserById godoc
// @Summary      Busca o perfil de um usuário por id
// @Description  Retorna id, name, description, email (quando visível) e configVisibility. O próprio usuário e ADMIN recebem o perfil completo; os demais recebem somente as entradas com shareWithCommunity=true e o email só se estiver compartilhado. A senha nunca é retornada.
// @Tags         users
// @Produce      json
// @Param        userId  path  string  true  "ID do usuário"
// @Success      200   {object}  userdto.UserProfileDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/{userId} [get]
func (u *userController) GetUserById() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId := c.Params("userId")
		if !uuidv7.IsValidString(userId) {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		requesterId := c.Locals(middleware.UserIdKey).(string)
		user, err := u.userService.GetUserById(userId, requesterId)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		return c.Status(fiber.StatusOK).JSON(userdto.UserProfileDtoOut{}.FromDomain(user))
	}
}

// UpdateUser godoc
// @Summary      Altera o perfil de um usuário
// @Description  Atualiza nome, resumo (description) e a configuração de visibilidade (configVisibility) do perfil. E-mail, senha e emailVerified não são alteráveis por este endpoint; o value da chave email da configVisibility é gerenciado pelo sistema. Somente o próprio usuário (id do token) ou um admin podem executar.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        userId  path  string  true  "ID do usuário"
// @Param        user    body  userdto.UpdateUserDtoIn  true  "Campos do perfil a alterar"
// @Success      200   {object}  userdto.UserDtoOut
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/{userId} [put]
func (u *userController) UpdateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId := c.Params("userId")
		if !uuidv7.IsValidString(userId) {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		var updateUserDto userdto.UpdateUserDtoIn
		if err := c.BodyParser(&updateUserDto); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		requesterId := c.Locals(middleware.UserIdKey).(string)
		updated, err := u.userService.UpdateUser(userId, requesterId, updateUserDto.ToDomain())
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		var updateUserDtoOut userdto.UserDtoOut
		updateUserDtoOut = *updateUserDtoOut.FromDomainUserWithProfile(updated)
		return c.Status(fiber.StatusOK).JSON(updateUserDtoOut)
	}
}

// DeleteUser godoc
// @Summary      Remove usuário (soft delete)
// @Description  Arquiva o usuário marcando deleted_at. Somente admins podem executar. O alvo não pode ser dono de comunidade/evento ativos nem ter participação ativa ou membership ativa.
// @Tags         users
// @Param        userId  path  string  true  "ID do usuário"
// @Success      204
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/user/{userId} [delete]
func (u *userController) DeleteUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId := c.Params("userId")
		if !uuidv7.IsValidString(userId) {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid params",
				[]rest_err.Causes{{Field: "userId", Message: "userId must be a valid UUID v7"}}))
		}
		requesterId := c.Locals(middleware.UserIdKey).(string)
		if err := u.userService.DeleteUser(userId, requesterId); err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}
