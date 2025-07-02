package controller_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller"
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
	db                *gorm.DB
	cleanupDB         func()
	userRepository    repository.UserRepository
	addressRepository repository.AddressRepository
	communityRepository repository.CommunityRepository
	testEmail         = "teste@ajuda.dev"
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
	db.AutoMigrate(&entity.UserEntity{}, &entity.AddressEntity{}, &entity.CommunityEntity{})

	return db, cleanup, nil
}

func TestMain(m *testing.M) {
	var err error

	db, cleanupDB, err = setupTestDB(context.Background())
	if err != nil {
		panic("Erro ao configurar o banco de dados: " + err.Error())
	}
	userRepository = repository.NewUserRepository(db)
	addressRepository = repository.NewAddressRepository(db)
	communityRepository = repository.NewCommunityRepository(db)
	code := m.Run()
	cleanupDB()
	os.Exit(code)
}

func setupApp() *fiber.App {
	app := fiber.New()
	userService :=  service.NewUserService(userRepository, validator.NewUserValidator());
	addressService := service.NewAddressService(addressRepository, validator.NewAddressValidator(), NewAddressSearchClient());
	routes.SetupRoutesUser(app, controller.NewUserController(userService))
	routes.SetupRoutesAddress(app, controller.NewAddressController(addressService))
	routes.SetupRoutesCommunities(app, controller.NewCommunityController(service.NewCommunityService(userService, addressService,communityRepository, validator.NewCommunityValidator())))

	return app
}

func cleanUsersTable() {
	db.Exec("DELETE FROM users")
}

func cleanAddressesTable() {
	db.Exec("DELETE FROM addresses")
}

func cleanCommunityTable(){
	db.Exec("DELETE FROM communities")
}

type addressSearchClientMock struct {
}

// SearchAddress implements client.AddressSearchClient.
func (a *addressSearchClientMock) SearchAddress(address domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	 if(address.ZipCode == "test_zip_code") {
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
