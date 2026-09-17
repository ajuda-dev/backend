package controller_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/ajuda-dev/backend/src/client/email"
	"github.com/ajuda-dev/backend/src/client/oauth"
	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/quota"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	addressctrl "github.com/ajuda-dev/backend/src/controller/address"
	communityctrl "github.com/ajuda-dev/backend/src/controller/community"
	eventctrl "github.com/ajuda-dev/backend/src/controller/event"
	identityctrl "github.com/ajuda-dev/backend/src/controller/identity"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	notificationctrl "github.com/ajuda-dev/backend/src/controller/notification"
	"github.com/ajuda-dev/backend/src/controller/routes"
	skillctrl "github.com/ajuda-dev/backend/src/controller/skill"
	addressentity "github.com/ajuda-dev/backend/src/data/address/entity"
	addressrepo "github.com/ajuda-dev/backend/src/data/address/repository"
	communityentity "github.com/ajuda-dev/backend/src/data/community/entity"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	skillentity "github.com/ajuda-dev/backend/src/data/skill/entity"
	skillrepo "github.com/ajuda-dev/backend/src/data/skill/repository"
	"github.com/ajuda-dev/backend/src/service/address"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	addressvalidator "github.com/ajuda-dev/backend/src/service/address/validator"
	"github.com/ajuda-dev/backend/src/service/community"
	communityvalidator "github.com/ajuda-dev/backend/src/service/community/validator"
	"github.com/ajuda-dev/backend/src/service/event"
	eventvalidator "github.com/ajuda-dev/backend/src/service/event/validator"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	identityvalidator "github.com/ajuda-dev/backend/src/service/identity/validator"
	"github.com/ajuda-dev/backend/src/service/notification"
	"github.com/ajuda-dev/backend/src/service/skill"
	skillvalidator "github.com/ajuda-dev/backend/src/service/skill/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db                      *gorm.DB
	cleanupDB               func()
	userRepository          userrepo.UserRepository
	addressRepository       addressrepo.AddressRepository
	communityRepository     communityrepo.CommunityRepository
	eventRepository         eventrepo.EventRepository
	eventUserRepository     eventrepo.EventUserRepository
	skillRepository         skillrepo.SkillRepository
	skillUserRepository     skillrepo.SkillUserRepository
	communityUserRepository communityrepo.CommunityUserRepository
	oauthAccountRepository  userrepo.OAuthAccountRepository
	outboxEventRepository   notificationrepo.OutboxEventRepository
	emailCodeRepository     userrepo.EmailCodeRepository
	testEmail               = "teste@ajuda.dev"
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
	// A busca por cidade/nome usa unaccent (plano 16), e este setup não passa pelo
	// database.Connect(): a extensão precisa ser criada também nos testes.
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS unaccent").Error; err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("error enabling unaccent extension: %w", err)
	}
	db.AutoMigrate(&userentity.UserEntity{}, &addressentity.AddressEntity{}, &communityentity.CommunityEntity{}, &evententity.EventEntity{}, &evententity.EventUserEntity{}, &skillentity.SkillEntity{}, &skillentity.SkillUserEntity{}, &communityentity.CommunityUserEntity{}, &userentity.OAuthAccountEntity{}, &notificationentity.OutboxEventEntity{}, &userentity.EmailCodeEntity{})
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_outbox_events_user_read_created ON outbox_events (user_id, read_at, created_at)").Error; err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("error creating outbox inbox index: %w", err)
	}

	return db, cleanup, nil
}

func TestMain(m *testing.M) {
	var err error

	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_EXPIRATION_TIME", "24")
	os.Setenv("OUTBOX_ENABLED", "false")
	// Generous defaults so existing suites are not blocked by production quotas/rates.
	os.Setenv("MAX_OWNED_COMMUNITIES", "1000")
	os.Setenv("MAX_OWNED_COMMUNITIES_MODERATOR", "1000")
	os.Setenv("MAX_PENDING_EVENTS", "1000")
	os.Setenv("MAX_PENDING_EVENTS_MODERATOR", "1000")
	os.Setenv("MAX_ACTIVE_EVENTS", "1000")
	os.Setenv("MAX_ACTIVE_EVENTS_MODERATOR", "1000")
	os.Setenv("MAX_COMMUNITY_MEMBERSHIPS", "1000")
	os.Setenv("MAX_COMMUNITY_MEMBERSHIPS_MODERATOR", "1000")
	os.Setenv("RATE_LIMIT_EVENT_CREATE_PER_HOUR", "1000")
	os.Setenv("RATE_LIMIT_EVENT_CREATE_PER_HOUR_MODERATOR", "1000")
	os.Setenv("RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR", "1000")
	os.Setenv("RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR_MODERATOR", "1000")
	os.Setenv("RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR", "1000")
	os.Setenv("RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR_MODERATOR", "1000")
	os.Unsetenv("MAX_SKILLS_PER_USER")
	os.Unsetenv("MAX_SKILLS_PER_USER_MODERATOR")

	db, cleanupDB, err = setupTestDB(context.Background())
	if err != nil {
		panic("Erro ao configurar o banco de dados: " + err.Error())
	}
	userRepository = userrepo.NewUserRepository(db, nil)
	addressRepository = addressrepo.NewAddressRepository(db)
	communityRepository = communityrepo.NewCommunityRepository(db)
	outboxEventRepository = notificationrepo.NewOutboxEventRepository(db)
	emailCodeRepository = userrepo.NewEmailCodeRepository(db)
	eventRepository = eventrepo.NewEventRepository(db, outboxEventRepository)
	eventUserRepository = eventrepo.NewEventUserRepository(db, outboxEventRepository)
	skillRepository = skillrepo.NewSkillRepository(db)
	skillUserRepository = skillrepo.NewSkillUserRepository(db)
	communityUserRepository = communityrepo.NewCommunityUserRepository(db)
	oauthAccountRepository = userrepo.NewOAuthAccountRepository(db)
	code := m.Run()
	cleanupDB()
	os.Exit(code)
}

func setupApp() *fiber.App {
	return setupAppWithEmail(email.NewNoopSender())
}

func setupAppWithEmail(sender email.EmailSender) *fiber.App {
	app := fiber.New()
	userRepo := userrepo.NewUserRepository(db, outboxEventRepository)
	authService := identity.NewAuthService(userRepo, sender)
	oauthService := identity.NewOAuthService(oauth.NewRegistry(oauth.ProvidersFromEnv()...), oauthAccountRepository, userRepo, authService)
	authMiddleware := middleware.VerifyJWT(authService)
	userService := identity.NewUserService(userRepo, identityvalidator.NewUserValidator(), authService,
		communityRepository, eventRepository, eventUserRepository, communityUserRepository,
		outboxEventRepository, emailCodeRepository)
	addressService := address.NewAddressService(addressRepository, addressvalidator.NewAddressValidator(), NewAddressSearchClient())
	routes.SetupRoutesUser(app, identityctrl.NewUserController(userService), identityctrl.NewAuthController(authService), authMiddleware)
	routes.SetupRoutesAuth(app, identityctrl.NewOAuthController(oauthService))
	routes.SetupRoutesAddress(app, addressctrl.NewAddressController(addressService), authMiddleware)
	quotaCfg := quota.LoadFromEnv()
	rateLimiter := quota.NewHourlyLimiter()
	communityService := community.NewCommunityService(userService, addressService, communityRepository, communityUserRepository, communityvalidator.NewCommunityValidator(), quotaCfg, rateLimiter)
	routes.SetupRoutesCommunities(app, communityctrl.NewCommunityController(communityService), authMiddleware)
	routes.SetupRoutesCommunityUsers(app, communityctrl.NewCommunityUserController(community.NewCommunityUserService(userService, communityService, communityUserRepository, quotaCfg, rateLimiter)), authMiddleware)
	eventService := event.NewEventService(userService, addressService, communityRepository, eventRepository, eventUserRepository, communityUserRepository, eventvalidator.NewEventValidator(), quotaCfg, rateLimiter)
	routes.SetupRoutesEvents(app, eventctrl.NewEventController(eventService), authMiddleware)
	routes.SetupRoutesEventUsers(app, eventctrl.NewEventUserController(event.NewEventUserService(userService, eventService, eventUserRepository, eventvalidator.NewEventUserValidator())), authMiddleware)
	skillService := skill.NewSkillService(userService, skillRepository, skillvalidator.NewSkillValidator())
	routes.SetupRoutesSkills(app, skillctrl.NewSkillController(skillService), authMiddleware)
	routes.SetupRoutesSkillUsers(app, skillctrl.NewSkillUserController(skill.NewSkillUserService(userService, skillService, skillUserRepository, skillvalidator.NewSkillUserValidator(), quotaCfg)), authMiddleware)
	routes.SetupRoutesNotifications(app, notificationctrl.NewNotificationController(notification.NewNotificationHub(), notification.NewNotificationService(outboxEventRepository)), authMiddleware)
	routes.SetupSwaggerRoute(app)

	return app
}

func cleanUsersTable() {
	db.Exec("DELETE FROM email_codes")
	db.Exec("DELETE FROM outbox_events")
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

func cleanOAuthAccountsTable() {
	db.Exec("DELETE FROM oauth_accounts")
}

type addressSearchClientMock struct {
}

// SearchAddress implements client.AddressSearchClient.
func (a *addressSearchClientMock) SearchAddress(address addressdomain.AddressDomain) (*addressdomain.AddressDomain, *rest_err.RestErr) {
	if address.ZipCode == "test_zip_code" {
		return &addressdomain.AddressDomain{
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
	token, restErr := identity.NewAuthService(userRepository, email.NewNoopSender()).
		CreateToken(&userdomain.UserDomain{Id: userId})
	if restErr != nil {
		t.Fatalf("failed to create token: %v", restErr)
	}
	return token
}

func createUserWithRole(t *testing.T, email string, role string) *userdomain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
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

func sessionCookieFrom(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == middleware.SessionCookieName {
			return cookie
		}
	}
	return nil
}
