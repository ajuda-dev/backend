package config

import (
	"log"

	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/database"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller"
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
	userService := initUserService(repository.NewUserRepository(db), service.NewAuthService())
	addressService := initAddressService(repository.NewAddressRepository(db))
	routes.SetupRoutesUser(app, initUserController(userService))
	routes.SetupRoutesAddress(app, initAddressController(addressService))
	routes.SetupRoutesCommunities(app, initCommunityController(repository.NewCommunityRepository(db), addressService, userService))
	routes.SetupSwaggerRoute(app)
	log.Fatal(app.Listen(":8080"))
}

func initUserController(userService service.UserService) controller.UserController {
	return controller.NewUserController(userService)
}

func initAddressController(addressService service.AddressService) controller.AddressController {
	return controller.NewAddressController(addressService)
}

func initCommunityController(
	communityRepository repository.CommunityRepository,
	addressService service.AddressService,
	userService service.UserService) controller.CommunityController {
	return controller.NewCommunityController(service.NewCommunityService(userService, addressService, communityRepository, validator.NewCommunityValidator()))
}

func initUserService(userRepository repository.UserRepository, authService service.AuthService) service.UserService {
	return service.NewUserService(userRepository, validator.NewUserValidator(), authService)
}

func initAddressService(addressRepository service.AddressService) service.AddressService {
	return service.NewAddressService(
		addressRepository,
		validator.NewAddressValidator(),
		client.NewAddressSearchClient())
}
