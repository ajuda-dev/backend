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


func SetupRoutesCommunities(app *fiber.App, communityController controller.CommunityController){
	communities := app.Group("/v1/community")
	communities.Post("/register", communityController.RegisterCommunity())
	communities.Get("", communityController.GetAllCommunities())
}