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
	user.Post("/logout", userController.Logout())
	user.Post("/forgot-password", authController.ForgotPassword())
	user.Post("/reset-password", authController.ResetPassword())
	user.Post("/verify-email", auth, userController.VerifyEmail())
	user.Post("/resend-verification", auth, userController.ResendVerification())
	user.Get("/me", auth, userController.Me())
	user.Get("", auth, userController.GetAllUsers())
	user.Get("/:userId", auth, userController.GetUserById())
	user.Put("/:userId", auth, userController.UpdateUser())
	user.Delete("/:userId", auth, userController.DeleteUser())
}

func SetupRoutesAuth(app *fiber.App, oauthController controller.OAuthController) {
	auth := app.Group("/v1/auth")
	// Rotas públicas por natureza: são o próprio fluxo de login, o usuário ainda não possui token.
	auth.Get("/:provider/login", oauthController.StartLogin())
	auth.Get("/:provider/callback", oauthController.Callback())
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
	communities.Get("/:id/members", communityUserController.GetCommunityMembers())

	users := app.Group("/v1/user")
	users.Get("/:userId/communities", auth, communityUserController.GetUserCommunities())
}

func SetupRoutesEvents(app *fiber.App, eventController controller.EventController, auth fiber.Handler) {
	events := app.Group("/v1/event", auth)
	events.Post("/register", eventController.RegisterEvent())
	events.Get("/:id", eventController.GetEventById())
	events.Get("", eventController.GetAllEvents())
	events.Put("/:id/approval", eventController.UpdateEventApproval())
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

func SetupRoutesNotifications(app *fiber.App, notificationController controller.NotificationController, auth fiber.Handler) {
	notifications := app.Group("/v1/notifications", auth)
	notifications.Get("/stream", notificationController.Stream())
	notifications.Get("", notificationController.List())
	notifications.Put("/:id/read", notificationController.MarkRead())
}

func SetupSwaggerRoute(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
