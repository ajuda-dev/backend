package config

import (
	"log"

	"github.com/ajuda-dev/backend/src/config/database"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/repository"
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
	log.Fatal(app.Listen(":8080"))
}


func initUserController(database *gorm.DB) controller.UserController {
	userRepository := repository.NewUserRepository(database)
	userService := service.NewUserService(userRepository, validator.NewUserValidator())
	return controller.NewUserController(userService)
}