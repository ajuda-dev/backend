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
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).
				JSON(rest_err.NewUnauthorizedError("missing or invalid Authorization header"))
		}
		token := strings.TrimPrefix(header, "Bearer ")
		userId, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(err.Code).JSON(err)
		}
		c.Locals(UserIdKey, userId)
		return c.Next()
	}
}
