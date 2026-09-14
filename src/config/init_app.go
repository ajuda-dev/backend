package config

import (
	"log"

	"github.com/ajuda-dev/backend/src/client/oauth"
	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/database"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/ajuda-dev/backend/src/service/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func InitApp() {
	logger.Info("iniciando aplicação")
	godotenv.Load()
	db, err := database.Connect()
	if err != nil {
		logger.Error("Failed to connect to the database", err)
	}
	app := fiber.New()
	userRepository := repository.NewUserRepository(db)
	addressRepository := repository.NewAddressRepository(db)
	communityRepository := repository.NewCommunityRepository(db)
	eventRepository := repository.NewEventRepository(db)
	eventUserRepository := repository.NewEventUserRepository(db)
	skillRepository := repository.NewSkillRepository(db)
	skillUserRepository := repository.NewSkillUserRepository(db)
	communityUserRepository := repository.NewCommunityUserRepository(db)

	authService := service.NewAuthService(userRepository)
	oauthService := service.NewOAuthService(oauth.NewRegistry(oauth.ProvidersFromEnv()...), repository.NewOAuthAccountRepository(db), userRepository, authService)
	userService := initUserService(userRepository, authService, communityRepository, eventRepository, eventUserRepository, communityUserRepository)
	addressService := initAddressService(addressRepository)
	eventService := service.NewEventService(
		userService,
		addressService,
		communityRepository,
		eventRepository,
		eventUserRepository,
		communityUserRepository,
		validator.NewEventValidator())
	skillService := service.NewSkillService(userService, skillRepository, validator.NewSkillValidator())
	authMiddleware := middleware.VerifyJWT(authService)
	communityService := service.NewCommunityService(userService, addressService, communityRepository, communityUserRepository, validator.NewCommunityValidator())
	communityUserService := service.NewCommunityUserService(userService, communityService, communityUserRepository)
	routes.SetupRoutesUser(app, initUserController(userService), initAuthController(authService), authMiddleware)
	routes.SetupRoutesAuth(app, controller.NewOAuthController(oauthService))
	routes.SetupRoutesAddress(app, initAddressController(addressService), authMiddleware)
	routes.SetupRoutesCommunities(app, controller.NewCommunityController(communityService), authMiddleware)
	routes.SetupRoutesCommunityUsers(app, controller.NewCommunityUserController(communityUserService), authMiddleware)
	routes.SetupRoutesEvents(app, controller.NewEventController(eventService), authMiddleware)
	routes.SetupRoutesEventUsers(app, initEventUserController(userService, eventService, eventUserRepository), authMiddleware)
	routes.SetupRoutesSkills(app, controller.NewSkillController(skillService), authMiddleware)
	routes.SetupRoutesSkillUsers(app, initSkillUserController(userService, skillService, skillUserRepository), authMiddleware)
	routes.SetupSwaggerRoute(app)
	log.Fatal(app.Listen(":8080"))
}

func initUserController(userService service.UserService) controller.UserController {
	return controller.NewUserController(userService)
}

func initAuthController(authService service.AuthService) controller.AuthController {
	return controller.NewAuthController(authService)
}

func initAddressController(addressService service.AddressService) controller.AddressController {
	return controller.NewAddressController(addressService)
}

func initEventUserController(
	userService service.UserService,
	eventService service.EventService,
	eventUserRepository repository.EventUserRepository) controller.EventUserController {
	return controller.NewEventUserController(service.NewEventUserService(userService, eventService, eventUserRepository, validator.NewEventUserValidator()))
}

func initSkillUserController(
	userService service.UserService,
	skillService service.SkillService,
	skillUserRepository repository.SkillUserRepository) controller.SkillUserController {
	return controller.NewSkillUserController(service.NewSkillUserService(userService, skillService, skillUserRepository, validator.NewSkillUserValidator()))
}

func initUserService(userRepository repository.UserRepository,
	authService service.AuthService,
	communityRepository repository.CommunityRepository,
	eventRepository repository.EventRepository,
	eventUserRepository repository.EventUserRepository,
	communityUserRepository repository.CommunityUserRepository) service.UserService {
	return service.NewUserService(userRepository, validator.NewUserValidator(), authService,
		communityRepository, eventRepository, eventUserRepository, communityUserRepository)
}

func initAddressService(addressRepository repository.AddressRepository) service.AddressService {
	return service.NewAddressService(
		addressRepository,
		validator.NewAddressValidator(),
		client.NewAddressSearchClient())
}
