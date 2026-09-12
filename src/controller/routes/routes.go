package routes

import (
	_ "github.com/ajuda-dev/backend/docs"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func SetupRoutesUser(app *fiber.App, userController controller.UserController, authController controller.AuthController, auth fiber.Handler) {

	user := app.Group("/v1/user")
	user.Post("/register", userController.RegisterUser())
	user.Post("/login", authController.LoginUser())
	user.Get("", auth, userController.GetAllUsers())
	user.Put("/:userId", auth, userController.UpdateUser())
	user.Delete("/:userId", auth, userController.DeleteUser())
}

func SetupRoutesAddress(app *fiber.App, addressController controller.AddressController, auth fiber.Handler) {
	address := app.Group("/v1/address", auth)
	address.Post("/register", addressController.RegisterAddress())
	address.Get("", addressController.GetAllAddresses())
}

func SetupRoutesCommunities(app *fiber.App, communityController controller.CommunityController, auth fiber.Handler) {
	communities := app.Group("/v1/community", auth)
	communities.Post("/register", communityController.RegisterCommunity())
	communities.Get("/:id", communityController.GetCommunityById())
	communities.Get("", communityController.GetAllCommunities())
	communities.Put("/:id", communityController.UpdateCommunity())
	communities.Delete("/:id", communityController.DeleteCommunity())
}

func SetupRoutesCommunityUsers(app *fiber.App, communityUserController controller.CommunityUserController, auth fiber.Handler) {
	communities := app.Group("/v1/community", auth)
	communities.Post("/:id/join", communityUserController.JoinCommunity())
	communities.Delete("/:id/leave", communityUserController.LeaveCommunity())
}

func SetupRoutesEvents(app *fiber.App, eventController controller.EventController, auth fiber.Handler) {
	events := app.Group("/v1/event", auth)
	events.Post("/register", eventController.RegisterEvent())
	events.Get("/:id", eventController.GetEventById())
	events.Get("", eventController.GetAllEvents())
	events.Delete("/:id", eventController.DeleteEventById())
}

func SetupRoutesEventUsers(app *fiber.App, eventUserController controller.EventUserController, auth fiber.Handler) {
	events := app.Group("/v1/event", auth)
	events.Post("/:eventId/join", eventUserController.JoinEvent())
	events.Post("/:eventId/participants", eventUserController.AddParticipant())
	events.Get("/:eventId/participants", eventUserController.GetParticipants())
	events.Put("/:eventId/participants/:userId/status", eventUserController.UpdateParticipantStatus())
	events.Delete("/:eventId/participants/:userId", eventUserController.CancelParticipation())
}

func SetupRoutesSkills(app *fiber.App, skillController controller.SkillController, auth fiber.Handler) {
	skills := app.Group("/v1/skill", auth)
	skills.Post("/register", skillController.RegisterSkill())
	skills.Get("/:id", skillController.GetSkillById())
	skills.Get("", skillController.GetAllSkills())
	skills.Put("/:id", skillController.UpdateSkill())
	skills.Delete("/:id", skillController.DeleteSkill())
}

func SetupRoutesSkillUsers(app *fiber.App, skillUserController controller.SkillUserController, auth fiber.Handler) {
	skills := app.Group("/v1/skill", auth)
	skills.Post("/:skillId/users", skillUserController.AssignSkill())

	users := app.Group("/v1/user")
	users.Get("/:userId/skills", auth, skillUserController.GetUserSkills())
	users.Delete("/:userId/skills/:skillId", auth, skillUserController.RemoveSkillFromUser())
}

func SetupSwaggerRoute(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
