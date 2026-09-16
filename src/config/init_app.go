package config

import (
	"log"

	"github.com/ajuda-dev/backend/src/client/email"
	"github.com/ajuda-dev/backend/src/client/oauth"
	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/database"
	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/config/logger"
	addressctrl "github.com/ajuda-dev/backend/src/controller/address"
	communityctrl "github.com/ajuda-dev/backend/src/controller/community"
	eventctrl "github.com/ajuda-dev/backend/src/controller/event"
	identityctrl "github.com/ajuda-dev/backend/src/controller/identity"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	notificationctrl "github.com/ajuda-dev/backend/src/controller/notification"
	"github.com/ajuda-dev/backend/src/controller/routes"
	skillctrl "github.com/ajuda-dev/backend/src/controller/skill"
	addressrepo "github.com/ajuda-dev/backend/src/data/address/repository"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	skillrepo "github.com/ajuda-dev/backend/src/data/skill/repository"
	"github.com/ajuda-dev/backend/src/service/address"
	addressvalidator "github.com/ajuda-dev/backend/src/service/address/validator"
	"github.com/ajuda-dev/backend/src/service/community"
	communityvalidator "github.com/ajuda-dev/backend/src/service/community/validator"
	"github.com/ajuda-dev/backend/src/service/event"
	eventvalidator "github.com/ajuda-dev/backend/src/service/event/validator"
	"github.com/ajuda-dev/backend/src/service/identity"
	identityvalidator "github.com/ajuda-dev/backend/src/service/identity/validator"
	"github.com/ajuda-dev/backend/src/service/notification"
	"github.com/ajuda-dev/backend/src/service/skill"
	skillvalidator "github.com/ajuda-dev/backend/src/service/skill/validator"
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
	outboxEventRepository := notificationrepo.NewOutboxEventRepository(db)
	emailCodeRepository := userrepo.NewEmailCodeRepository(db)
	emailSender := email.FromEnv()
	userRepository := userrepo.NewUserRepository(db, outboxEventRepository)
	addressRepository := addressrepo.NewAddressRepository(db)
	communityRepository := communityrepo.NewCommunityRepository(db)
	eventRepository := eventrepo.NewEventRepository(db, outboxEventRepository)
	eventUserRepository := eventrepo.NewEventUserRepository(db, outboxEventRepository)
	skillRepository := skillrepo.NewSkillRepository(db)
	skillUserRepository := skillrepo.NewSkillUserRepository(db)
	communityUserRepository := communityrepo.NewCommunityUserRepository(db)

	authService := identity.NewAuthService(userRepository, emailSender)
	oauthService := identity.NewOAuthService(oauth.NewRegistry(oauth.ProvidersFromEnv()...), userrepo.NewOAuthAccountRepository(db), userRepository, authService)
	userService := initUserService(userRepository, authService, communityRepository, eventRepository, eventUserRepository, communityUserRepository, outboxEventRepository, emailCodeRepository)
	addressService := initAddressService(addressRepository)
	eventService := event.NewEventService(
		userService,
		addressService,
		communityRepository,
		eventRepository,
		eventUserRepository,
		communityUserRepository,
		eventvalidator.NewEventValidator())
	skillService := skill.NewSkillService(userService, skillRepository, skillvalidator.NewSkillValidator())
	authMiddleware := middleware.VerifyJWT(authService)
	communityService := community.NewCommunityService(userService, addressService, communityRepository, communityUserRepository, communityvalidator.NewCommunityValidator())
	communityUserService := community.NewCommunityUserService(userService, communityService, communityUserRepository)
	routes.SetupRoutesUser(app, initUserController(userService), initAuthController(authService), authMiddleware)
	routes.SetupRoutesAuth(app, identityctrl.NewOAuthController(oauthService))
	routes.SetupRoutesAddress(app, initAddressController(addressService), authMiddleware)
	routes.SetupRoutesCommunities(app, communityctrl.NewCommunityController(communityService), authMiddleware)
	routes.SetupRoutesCommunityUsers(app, communityctrl.NewCommunityUserController(communityUserService), authMiddleware)
	routes.SetupRoutesEvents(app, eventctrl.NewEventController(eventService), authMiddleware)
	routes.SetupRoutesEventUsers(app, initEventUserController(userService, eventService, eventUserRepository), authMiddleware)
	routes.SetupRoutesSkills(app, skillctrl.NewSkillController(skillService), authMiddleware)
	routes.SetupRoutesSkillUsers(app, initSkillUserController(userService, skillService, skillUserRepository), authMiddleware)
	hub := notification.NewNotificationHub()
	routes.SetupRoutesNotifications(app, notificationctrl.NewNotificationController(hub, notification.NewNotificationService(outboxEventRepository)), authMiddleware)
	routes.SetupSwaggerRoute(app)
	job.StartOutboxJob(db, job.OutboxConfigFromEnv(), job.NewDispatchingOutboxHandler(
		job.NewSSEOutboxHandler(hub),
		identity.NewCreatedAccountHandler(userRepository, emailCodeRepository, emailSender, identity.EmailCodeConfigFromEnv()),
	))
	log.Fatal(app.Listen(":8080"))
}

func initUserController(userService identity.UserService) identityctrl.UserController {
	return identityctrl.NewUserController(userService)
}

func initAuthController(authService identity.AuthService) identityctrl.AuthController {
	return identityctrl.NewAuthController(authService)
}

func initAddressController(addressService address.AddressService) addressctrl.AddressController {
	return addressctrl.NewAddressController(addressService)
}

func initEventUserController(
	userService identity.UserService,
	eventService event.EventService,
	eventUserRepository eventrepo.EventUserRepository) eventctrl.EventUserController {
	return eventctrl.NewEventUserController(event.NewEventUserService(userService, eventService, eventUserRepository, eventvalidator.NewEventUserValidator()))
}

func initSkillUserController(
	userService identity.UserService,
	skillService skill.SkillService,
	skillUserRepository skillrepo.SkillUserRepository) skillctrl.SkillUserController {
	return skillctrl.NewSkillUserController(skill.NewSkillUserService(userService, skillService, skillUserRepository, skillvalidator.NewSkillUserValidator()))
}

func initUserService(userRepository userrepo.UserRepository,
	authService identity.AuthService,
	communityRepository communityrepo.CommunityRepository,
	eventRepository eventrepo.EventRepository,
	eventUserRepository eventrepo.EventUserRepository,
	communityUserRepository communityrepo.CommunityUserRepository,
	outboxEventRepository notificationrepo.OutboxEventRepository,
	emailCodeRepository userrepo.EmailCodeRepository) identity.UserService {
	return identity.NewUserService(userRepository, identityvalidator.NewUserValidator(), authService,
		communityRepository, eventRepository, eventUserRepository, communityUserRepository,
		outboxEventRepository, emailCodeRepository)
}

func initAddressService(addressRepository addressrepo.AddressRepository) address.AddressService {
	return address.NewAddressService(
		addressRepository,
		addressvalidator.NewAddressValidator(),
		client.NewAddressSearchClient())
}
