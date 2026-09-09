package controller_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	"github.com/ajuda-dev/backend/src/controller/routes"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db                  *gorm.DB
	cleanupDB           func()
	userRepository      repository.UserRepository
	addressRepository   repository.AddressRepository
	communityRepository repository.CommunityRepository
	eventRepository     repository.EventRepository
	eventUserRepository repository.EventUserRepository
	skillRepository     repository.SkillRepository
	skillUserRepository repository.SkillUserRepository
	communityUserRepository repository.CommunityUserRepository
	testEmail           = "teste@ajuda.dev"
)

func setupTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "help_dev_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=help_dev_test sslmode=disable",
		host, port.Port())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, err
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Erro ao parar o container: %v", err)
		}
	}
	db.AutoMigrate(&entity.UserEntity{}, &entity.AddressEntity{}, &entity.CommunityEntity{}, &entity.EventEntity{}, &entity.EventUserEntity{}, &entity.SkillEntity{}, &entity.SkillUserEntity{}, &entity.CommunityUserEntity{})

	return db, cleanup, nil
}

func TestMain(m *testing.M) {
	var err error

	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_EXPIRATION_TIME", "24")

	db, cleanupDB, err = setupTestDB(context.Background())
	if err != nil {
		panic("Erro ao configurar o banco de dados: " + err.Error())
	}
	userRepository = repository.NewUserRepository(db)
	addressRepository = repository.NewAddressRepository(db)
	communityRepository = repository.NewCommunityRepository(db)
	eventRepository = repository.NewEventRepository(db)
	eventUserRepository = repository.NewEventUserRepository(db)
	skillRepository = repository.NewSkillRepository(db)
	skillUserRepository = repository.NewSkillUserRepository(db)
	communityUserRepository = repository.NewCommunityUserRepository(db)
	code := m.Run()
	cleanupDB()
	os.Exit(code)
}

func setupApp() *fiber.App {
	app := fiber.New()
	authService := service.NewAuthService(userRepository)
	authMiddleware := middleware.VerifyJWT(authService)
	userService := service.NewUserService(userRepository, validator.NewUserValidator(), authService,
		communityRepository, eventRepository, eventUserRepository, communityUserRepository)
	addressService := service.NewAddressService(addressRepository, validator.NewAddressValidator(), NewAddressSearchClient())
	routes.SetupRoutesUser(app, controller.NewUserController(userService), controller.NewAuthController(authService), authMiddleware)
	routes.SetupRoutesAddress(app, controller.NewAddressController(addressService), authMiddleware)
	communityService := service.NewCommunityService(userService, addressService, communityRepository, communityUserRepository, validator.NewCommunityValidator())
	routes.SetupRoutesCommunities(app, controller.NewCommunityController(communityService), authMiddleware)
	routes.SetupRoutesCommunityUsers(app, controller.NewCommunityUserController(service.NewCommunityUserService(userService, communityService, communityUserRepository)), authMiddleware)
	eventService := service.NewEventService(userService, addressService, communityRepository, eventRepository, eventUserRepository, validator.NewEventValidator())
	routes.SetupRoutesEvents(app, controller.NewEventController(eventService), authMiddleware)
	routes.SetupRoutesEventUsers(app, controller.NewEventUserController(service.NewEventUserService(userService, eventService, eventUserRepository, validator.NewEventUserValidator())), authMiddleware)
	skillService := service.NewSkillService(userService, skillRepository, validator.NewSkillValidator())
	routes.SetupRoutesSkills(app, controller.NewSkillController(skillService), authMiddleware)
	routes.SetupRoutesSkillUsers(app, controller.NewSkillUserController(service.NewSkillUserService(userService, skillService, skillUserRepository, validator.NewSkillUserValidator())), authMiddleware)
	routes.SetupSwaggerRoute(app)

	return app
}

func cleanUsersTable() {
	db.Exec("DELETE FROM users")
}

func cleanAddressesTable() {
	db.Exec("DELETE FROM addresses")
}

func cleanCommunityTable() {
	db.Exec("DELETE FROM community")
}

func cleanEventsTable() {
	db.Exec("DELETE FROM events")
}

func cleanEventUsersTable() {
	db.Exec("DELETE FROM event_users")
}

func cleanSkillsTable() {
	db.Exec("DELETE FROM skills")
}

func cleanSkillUsersTable() {
	db.Exec("DELETE FROM skill_users")
}

func cleanCommunityUsersTable() {
	db.Exec("DELETE FROM community_users")
}

type addressSearchClientMock struct {
}

// SearchAddress implements client.AddressSearchClient.
func (a *addressSearchClientMock) SearchAddress(address domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	if address.ZipCode == "test_zip_code" {
		return &domain.AddressDomain{
			City:    "Mock City",
			State:   "Mock State",
			Street:  "Mock Street",
			ZipCode: "12345-678",
		}, nil
	} else {
		return nil, rest_err.NewBadRequestError("Invalid address data")
	}
}

func NewAddressSearchClient() client.AddressSearchClient {
	return &addressSearchClientMock{}
}

func verifyCodeError(t *testing.T, respBody rest_err.RestErr) {

	if respBody.Err != "bad_request" {
		t.Errorf("esperava error 'bad_request', recebeu '%s'", respBody.Error())
	}
	if respBody.Code != 400 {
		t.Errorf("esperava code 400, recebeu %d", respBody.Code)
	}
}

func validTokenFor(t *testing.T, userId string) string {
	t.Helper()
	token, restErr := service.NewAuthService(userRepository).
		CreateToken(&domain.UserDomain{Id: userId})
	if restErr != nil {
		t.Fatalf("failed to create token: %v", restErr)
	}
	return token
}

func createUserWithRole(t *testing.T, email string, role string) *domain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "usuario " + role,
		Email:    email,
		Password: "123456",
		Role:     role,
	})
	if createErr != nil {
		t.Fatalf("failed to create user with role %s: %v", role, createErr)
	}
	if user.Role != role {
		t.Fatalf("esperava role %s persistida no usuário, recebeu %s", role, user.Role)
	}
	return user
}

func doAuthedRequest(app *fiber.App, req *http.Request, token string) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+token)
	return app.Test(req)
}
