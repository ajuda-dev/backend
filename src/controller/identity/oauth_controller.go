package identity

import (
	"crypto/subtle"
	"os"
	"strconv"
	"strings"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service/identity"
	"github.com/gofiber/fiber/v2"
)

const (
	oauthStateCookiePrefix  = "oauth_state_"
	oauthStateCookieMaxAge  = 600
	oauthFrontendURLEnv     = "OAUTH_FRONTEND_URL"
	oauthCookieSecureEnv    = "OAUTH_COOKIE_SECURE"
	oauthFrontendURLDefault = "http://localhost:3000"
	oauthSuccessPath        = "/auth/callback"
	oauthErrorQuery         = "?error=auth_failed"
)

func NewOAuthController(oauthService identity.OAuthService) OAuthController {
	return &oauthController{
		oauthService: oauthService,
	}
}

type OAuthController interface {
	StartLogin() fiber.Handler
	Callback() fiber.Handler
}

type oauthController struct {
	oauthService identity.OAuthService
}

// StartLogin godoc
// @Summary      Inicia o login via OAuth
// @Description  Redireciona o navegador para o provedor OAuth informado no path (ex.: github). Endpoint público: é o início do fluxo de login. O state é gravado em cookie HttpOnly de curta duração e conferido no callback.
// @Tags         auth
// @Produce      json
// @Param        provider  path  string  true  "Provedor OAuth (ex.: github)"
// @Success      302  {string}  string  "Redireciona para a página de autorização do provedor"
// @Failure      400  {object}  map[string]interface{}
// @Router       /v1/auth/{provider}/login [get]
func (o *oauthController) StartLogin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := c.Params("provider")
		authorizationURL, state, err := o.oauthService.LoginURL(provider)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		c.Cookie(oauthStateCookie(provider, state, oauthStateCookieMaxAge))
		return c.Redirect(authorizationURL, fiber.StatusFound)
	}
}

// Callback godoc
// @Summary      Callback do login via OAuth
// @Description  Recebe o code do provedor, autentica o usuário (reaproveitando o vínculo existente, casando pelo e-mail verificado ou criando um usuário novo sem senha) e grava o token JWT em cookie de sessão HttpOnly (ajudadev_session) antes de redirecionar para o frontend — o token não trafega na URL. Em falha, redireciona para o frontend com ?error=auth_failed. Endpoint público.
// @Tags         auth
// @Produce      json
// @Param        provider  path   string  true  "Provedor OAuth (ex.: github)"
// @Param        code      query  string  true  "Código de autorização devolvido pelo provedor"
// @Param        state     query  string  true  "State enviado no início do fluxo"
// @Success      302  {string}  string  "Redireciona para o frontend com o cookie de sessão gravado"
// @Failure      400  {object}  map[string]interface{}
// @Router       /v1/auth/{provider}/callback [get]
func (o *oauthController) Callback() fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := c.Params("provider")
		expectedState := c.Cookies(oauthStateCookiePrefix + provider)
		c.Cookie(oauthStateCookie(provider, "", -1))

		if !isValidOAuthState(c.Query("state"), expectedState) {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestError("invalid oauth state"))
		}

		_, token, err := o.oauthService.Authenticate(provider, c.Query("code"))
		if err != nil {
			logger.Error("error: ", err)
			return c.Redirect(oauthErrorRedirectURL(), fiber.StatusFound)
		}
		middleware.SetSessionCookie(c, token)
		return c.Redirect(oauthSuccessRedirectURL(), fiber.StatusFound)
	}
}

func oauthStateCookie(provider string, value string, maxAge int) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     oauthStateCookiePrefix + provider,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HTTPOnly: true,
		Secure:   oauthCookieSecure(),
		SameSite: fiber.CookieSameSiteLaxMode,
	}
}

func isValidOAuthState(state string, expectedState string) bool {
	if state == "" || expectedState == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) == 1
}

func oauthSuccessRedirectURL() string {
	return oauthFrontendURL() + oauthSuccessPath
}

func oauthErrorRedirectURL() string {
	return oauthFrontendURL() + oauthErrorQuery
}

func oauthFrontendURL() string {
	frontendURL := strings.TrimSpace(os.Getenv(oauthFrontendURLEnv))
	if frontendURL == "" {
		return oauthFrontendURLDefault
	}
	return strings.TrimSuffix(frontendURL, "/")
}

func oauthCookieSecure() bool {
	secure, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(oauthCookieSecureEnv)))
	if err != nil {
		return false
	}
	return secure
}
