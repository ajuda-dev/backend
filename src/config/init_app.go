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
	"gorm.io/gorm"
)

func InitApp() {
	logger.Info("iniciando aplicação")
	godotenv.Load()
	db, err := database.Connect()
	if err != nil {
		logger.Error("Failed to connect to the database", err)
	}
	app := fiber.New()
	routes.SetupRoutesUser(app, initUserController(db))
	routes.SetupRoutesAddress(app, initAddressController(db))
	log.Fatal(app.Listen(":8080"))
}

func initUserController(database *gorm.DB) controller.UserController {
	userRepository := repository.NewUserRepository(database)
	userService := service.NewUserService(userRepository, validator.NewUserValidator())
	return controller.NewUserController(userService)
}

func initAddressController(db *gorm.DB) controller.AddressController {
	addressRepository := repository.NewAddressRepository(db)
	addressService := service.NewAddressService(
		addressRepository,
		validator.NewAddressValidator(),
		client.NewAddressSearchClient())
	return controller.NewAddressController(addressService)
}
