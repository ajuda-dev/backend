package routes

import (
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutesUser(app *fiber.App, userController controller.UserController) {

	user := app.Group("/v1/user")
	user.Post("/register", userController.RegisterUser()) 
}


func SetupRoutesAddress(app *fiber.App, addressController controller.AddressController) {
	address := app.Group("/v1/address")
	address.Post("/register", addressController.RegisterAddress())
}