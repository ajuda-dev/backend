package routes

import (
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutesUser(app *fiber.App, userController controller.UserController) {

	user := app.Group("/v1/user")
	user.Post("/register", userController.RegisterUser()) 
}