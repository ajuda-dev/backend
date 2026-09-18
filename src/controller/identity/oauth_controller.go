package identity

import (
	"crypto/subtle"
	"encoding/json"
	"html"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/client/oauth"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/service/identity"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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
// @Description  Grava o state em cookie HttpOnly no mesmo host do callback e envia o navegador ao provedor OAuth. Se o login começar em outro host (ex.: localhost vs 127.0.0.1), redireciona antes para o host do callback. Endpoint público.
// @Tags         auth
// @Produce      html
// @Param        provider  path  string  true  "Provedor OAuth (ex.: github)"
// @Success      200  {string}  string  "HTML que redireciona para a página de autorização do provedor"
// @Success      302  {string}  string  "Redireciona para o host do callback quando o login começou em outro host"
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
		if dest, ok := oauthCanonicalLoginURL(c.Hostname(), c.Path(), string(c.Request().URI().QueryString()), oauth.CallbackURL(provider)); ok {
			return c.Redirect(dest, fiber.StatusFound)
		}
		c.Cookie(oauthStateCookie(provider, state, oauthStateCookieMaxAge))
		c.Type("html", "utf-8")
		return c.Status(fiber.StatusOK).SendString(oauthProviderRedirectHTML(authorizationURL))
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
			logger.Info("invalid oauth state",
				zap.String("provider", provider),
				zap.Bool("cookie_present", expectedState != ""),
				zap.Bool("state_present", c.Query("state") != ""),
				zap.String("host", c.Hostname()))
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
	cookie := &fiber.Cookie{
		Name:     oauthStateCookiePrefix + provider,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HTTPOnly: true,
		Secure:   oauthCookieSecure(),
		SameSite: fiber.CookieSameSiteLaxMode,
	}
	if maxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(maxAge) * time.Second)
	}
	return cookie
}

func oauthCanonicalLoginURL(requestHost, requestPath, requestQuery, callbackURL string) (string, bool) {
	requestHost = strings.TrimSpace(requestHost)
	callbackURL = strings.TrimSpace(callbackURL)
	if requestHost == "" || callbackURL == "" {
		return "", false
	}
	parsed, err := url.Parse(callbackURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if strings.EqualFold(requestHost, parsed.Host) {
		return "", false
	}
	next := url.URL{
		Scheme:   parsed.Scheme,
		Host:     parsed.Host,
		Path:     requestPath,
		RawQuery: requestQuery,
	}
	return next.String(), true
}

func oauthProviderRedirectHTML(authorizationURL string) string {
	href := html.EscapeString(authorizationURL)
	jsURL, err := json.Marshal(authorizationURL)
	if err != nil {
		jsURL = []byte(`""`)
	}
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><meta http-equiv="refresh" content="0;url=` + href + `"></head><body><a href="` + href + `">Continue</a><script>window.location.replace(` + string(jsURL) + `)</script></body></html>`
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
