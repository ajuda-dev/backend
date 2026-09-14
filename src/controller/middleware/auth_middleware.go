package middleware

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

const UserIdKey = "user_id"

func VerifyJWT(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, fromCookie := requestToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).
				JSON(rest_err.NewUnauthorizedError("missing or invalid Authorization header"))
		}
		userId, err := authService.ValidateToken(token)
		if err != nil {
			if fromCookie {
				ClearSessionCookie(c)
			}
			return c.Status(err.Code).JSON(err)
		}
		c.Locals(UserIdKey, userId)
		return c.Next()
	}
}

// Cookie de sessão (navegador) ou Authorization: Bearer (apps nativos e demais clientes de API).
func requestToken(c *fiber.Ctx) (string, bool) {
	if header := c.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		if token := strings.TrimPrefix(header, "Bearer "); token != "" {
			return token, false
		}
	}
	return c.Cookies(SessionCookieName), true
}
