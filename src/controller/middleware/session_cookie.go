package middleware

import (
	"os"
	"strconv"
	"strings"

	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

const (
	SessionCookieName      = "ajudadev_session"
	NativeClientHeader     = "X-Client-Type"
	nativeClientValue      = "native"
	sessionCookieSecureEnv = "SESSION_COOKIE_SECURE"
)

func SetSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(sessionCookie(token, int(service.JWTExpiration().Seconds())))
}

func ClearSessionCookie(c *fiber.Ctx) {
	c.Cookie(sessionCookie("", -1))
}

// Clientes nativos (app mobile/desktop) não leem cookie HttpOnly: declarando-se com este
// header recebem o token no corpo para guardar em secure storage. O navegador nunca o envia.
func WantsTokenInBody(c *fiber.Ctx) bool {
	return strings.EqualFold(strings.TrimSpace(c.Get(NativeClientHeader)), nativeClientValue)
}

// HttpOnly: o JavaScript do front nunca lê o token; o navegador envia o cookie sozinho.
func sessionCookie(token string, maxAge int) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HTTPOnly: true,
		Secure:   sessionCookieSecure(),
		SameSite: fiber.CookieSameSiteLaxMode,
	}
}

func sessionCookieSecure() bool {
	secure, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(sessionCookieSecureEnv)))
	if err != nil {
		return false
	}
	return secure
}
