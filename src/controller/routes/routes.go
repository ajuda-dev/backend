package routes

import (
	_ "github.com/ajuda-dev/backend/docs"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func SetupRoutesUser(app *fiber.App, userController controller.UserController, authController controller.AuthController) {

	user := app.Group("/v1/user")
	user.Post("/register", userController.RegisterUser())
	user.Post("/login", authController.LoginUser())
}

func SetupRoutesAddress(app *fiber.App, addressController controller.AddressController) {
	address := app.Group("/v1/address")
	address.Post("/register", addressController.RegisterAddress())
}

func SetupRoutesCommunities(app *fiber.App, communityController controller.CommunityController) {
	communities := app.Group("/v1/community")
	communities.Post("/register", communityController.RegisterCommunity())
	communities.Get("", communityController.GetAllCommunities())
}

func SetupRoutesEvents(app *fiber.App, eventController controller.EventController) {
	events := app.Group("/v1/event")
	events.Post("/register", eventController.RegisterEvent())
	events.Get("/:id", eventController.GetEventById())
	events.Get("", eventController.GetAllEvents())
	events.Delete("/:id", eventController.DeleteEventById())
}

func SetupSwaggerRoute(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
