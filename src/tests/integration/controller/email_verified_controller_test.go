package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func userEmailVerifiedAt(t *testing.T, userId string) *time.Time {
	t.Helper()
	var verifiedAt sql.NullTime
	if err := db.Raw("SELECT email_verified_at FROM users WHERE id = ?", userId).Scan(&verifiedAt).Error; err != nil {
		t.Fatalf("failed to read email_verified_at: %v", err)
	}
	if !verifiedAt.Valid {
		return nil
	}
	return &verifiedAt.Time
}

func clearEmailVerifiedAt(t *testing.T, userId string) {
	t.Helper()
	if err := db.Exec("UPDATE users SET email_verified_at = NULL WHERE id = ?", userId).Error; err != nil {
		t.Fatalf("failed to clear email_verified_at: %v", err)
	}
}

func TestRegisterReturnsEmailVerifiedFalse(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()

	resp, err := app.Test(newUserRegisterRequest([]byte(`{
		"name": "teste",
		"email": "verified_register@ajuda.dev",
		"password": "123456"
	}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("esperava 201 no register, recebeu %d", resp.StatusCode)
	}

	var respDto userdto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respDto.EmailVerified {
		t.Error("esperava emailVerified false no register")
	}

	user, findErr := userRepository.GetUserByEmail("verified_register@ajuda.dev")
	if findErr != nil {
		t.Fatalf("failed to find registered user: %v", findErr)
	}
	if user.EmailVerifiedAt != nil {
		t.Fatal("esperava email_verified_at null após o register")
	}
}

func TestLoginReturnsEmailVerifiedTrue(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithHashedPassword(t, app, "verified_login@ajuda.dev", userdomain.UserRoleUser, "123456")

	resp, err := app.Test(newUserLoginRequest([]byte(`{
		"email": "` + user.Email + `",
		"password": "123456"
	}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no login, recebeu %d", resp.StatusCode)
	}

	var respDto userdto.LoginUserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if !respDto.EmailVerified {
		t.Error("esperava emailVerified true no login")
	}
}

func TestMeReturnsEmailVerifiedTrue(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "verified_me@ajuda.dev", userdomain.UserRoleUser)

	resp, err := app.Test(newRequestWithSessionCookie(http.MethodGet, "/v1/user/me", validTokenFor(t, user.Id)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no /v1/user/me, recebeu %d", resp.StatusCode)
	}

	var respDto userdto.UserDtoOut
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if !respDto.EmailVerified {
		t.Error("esperava emailVerified true no /v1/user/me")
	}
}

func TestMeReturnsEmailVerifiedFalseWhenTimestampIsNull(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "unverified_me@ajuda.dev", userdomain.UserRoleUser)
	clearEmailVerifiedAt(t, user.Id)

	resp, err := app.Test(newRequestWithSessionCookie(http.MethodGet, "/v1/user/me", validTokenFor(t, user.Id)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no /v1/user/me, recebeu %d", resp.StatusCode)
	}

	body := decodeRawUserDtoOut(t, resp)
	verified, exists := body["emailVerified"]
	if !exists {
		t.Fatal("esperava a chave emailVerified no /v1/user/me")
	}
	if verified != false {
		t.Errorf("esperava emailVerified false com timestamp null, recebeu %v", verified)
	}
}

func TestLoginReturnsEmailVerifiedFalseWhenTimestampIsNull(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user, createErr := userRepository.CreateUser(&userdomain.UserDomain{
		Name:     "unverified login",
		Email:    "unverified_login@ajuda.dev",
		Password: string(hashedPassword),
		Role:     userdomain.UserRoleUser,
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	clearEmailVerifiedAt(t, user.Id)

	resp, err := app.Test(newUserLoginRequest([]byte(`{
		"email": "unverified_login@ajuda.dev",
		"password": "123456"
	}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no login, recebeu %d", resp.StatusCode)
	}

	body := decodeRawUserDtoOut(t, resp)
	if body["emailVerified"] != false {
		t.Errorf("esperava emailVerified false no login com timestamp null, recebeu %v", body["emailVerified"])
	}
}

func TestUpdateUserDoesNotChangeEmailVerifiedAt(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "verified_update@ajuda.dev", userdomain.UserRoleUser)
	before := userEmailVerifiedAt(t, user.Id)
	if before == nil {
		t.Fatal("esperava email_verified_at preenchido antes do update")
	}

	resp := doPutUser(t, app, user.Id, []byte(`{
		"name": "Nome Novo",
		"emailVerified": false
	}`), validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("esperava 200 no update, recebeu %d", resp.StatusCode)
	}
	respDto := decodeUserDtoOut(t, resp)
	if !respDto.EmailVerified {
		t.Error("esperava emailVerified true na resposta do update")
	}

	after := userEmailVerifiedAt(t, user.Id)
	if after == nil {
		t.Fatal("esperava email_verified_at preservado após o update")
	}
	if !after.Equal(*before) {
		t.Errorf("esperava email_verified_at intacto (antes %v, depois %v)", before, after)
	}
}

func TestGetUserProfileDoesNotReturnEmailVerified(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "verified_profile@ajuda.dev", userdomain.UserRoleUser)

	resp := doGetUserProfile(t, app, user.Id, validTokenFor(t, user.Id))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no perfil, recebeu %d", resp.StatusCode)
	}
	_, body := getUserProfileRawBody(t, resp)
	if _, exists := body["emailVerified"]; exists {
		t.Errorf("esperava o perfil sem emailVerified, recebeu %+v", body)
	}
}

func TestCreateUserWithoutOutboxSetsEmailVerifiedAt(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	user := createUserWithRole(t, "fixture_verified@ajuda.dev", userdomain.UserRoleUser)
	if user.EmailVerifiedAt == nil {
		t.Fatal("esperava email_verified_at preenchido no CreateUser sem outbox")
	}
}

func TestOAuthCreateUserSetsEmailVerifiedAt(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	_, state, stateCookie := startOAuthLogin(t, app, "github")
	response := completeOAuthCallback(t, app, "github", "valid-code", state, stateCookie)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no callback oauth, recebeu %d", response.StatusCode)
	}

	var userEntity userentity.UserEntity
	if err := db.Where("email = ?", fake.email).First(&userEntity).Error; err != nil {
		t.Fatalf("esperava usuário criado pelo oauth: %v", err)
	}
	if userEntity.EmailVerifiedAt == nil {
		t.Fatal("esperava email_verified_at preenchido no usuário criado via oauth")
	}
}

func TestOAuthCreateUserDoesNotInsertCreatedAccount(t *testing.T) {
	cleanUsersTable()
	cleanOAuthAccountsTable()
	fake := newFakeGithubServer(t)
	configureGithubProvider(t, fake)
	app := setupApp()

	_, state, stateCookie := startOAuthLogin(t, app, "github")
	response := completeOAuthCallback(t, app, "github", "valid-code", state, stateCookie)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("esperava 302 no callback oauth, recebeu %d", response.StatusCode)
	}

	var userEntity userentity.UserEntity
	if err := db.Where("email = ?", fake.email).First(&userEntity).Error; err != nil {
		t.Fatalf("esperava usuário criado pelo oauth: %v", err)
	}
	var count int64
	if err := db.Model(&notificationentity.OutboxEventEntity{}).
		Where("user_id = ? AND type = ?", userEntity.Id, notificationdomain.OutboxTypeCreatedAccount).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("esperava zero CREATED_ACCOUNT no oauth, recebeu %d", count)
	}
}
