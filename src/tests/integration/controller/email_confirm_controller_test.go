package controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/client/email"
	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/ajuda-dev/backend/src/service/notification"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type failingEmailSender struct {
	err error
}

func (f failingEmailSender) Send(_, _, _ string) error {
	return f.err
}

func processCreatedAccountOutbox(t *testing.T, sender email.EmailSender) {
	t.Helper()
	if sender == nil {
		sender = email.NewNoopSender()
	}
	handler := job.NewDispatchingOutboxHandler(
		job.NewSSEOutboxHandler(notification.NewNotificationHub()),
		identity.NewCreatedAccountHandler(userRepository, emailCodeRepository, sender, identity.EmailCodeConfigFromEnv()),
	)
	job.RunOutboxOnce(outboxEventRepository, handler, 50)
}

func registerConfirmUser(t *testing.T, app *fiber.App, addr string) (*http.Response, *userdomain.UserDomain) {
	t.Helper()
	resp, err := app.Test(newUserRegisterRequest([]byte(`{
		"name": "confirm user",
		"email": "` + addr + `",
		"password": "123456"
	}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	user, findErr := userRepository.GetUserByEmail(addr)
	require.Nil(t, findErr)
	require.NotNil(t, user)
	return resp, user
}

func createdAccountEvents(t *testing.T, userId string) []notificationentity.OutboxEventEntity {
	t.Helper()
	var events []notificationentity.OutboxEventEntity
	require.NoError(t, db.Where("user_id = ? AND type = ?", userId, notificationdomain.OutboxTypeCreatedAccount).
		Order("id ASC").Find(&events).Error)
	return events
}

func newVerifyEmailRequest(code, token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/user/verify-email", bytes.NewBufferString(`{"code":"`+code+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func newResendVerificationRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/user/resend-verification", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestRegisterInsertsCreatedAccountPending(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	addr := "confirm_pending_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)

	events := createdAccountEvents(t, user.Id)
	require.Len(t, events, 1)
	assert.Equal(t, notificationdomain.OutboxStatusPending, events[0].Status)
	assert.Nil(t, user.EmailVerifiedAt)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(events[0].Payload, &payload))
	assert.Equal(t, addr, payload["email"])
	assert.Equal(t, "confirm user", payload["name"])
	_, hasCode := payload["code"]
	assert.False(t, hasCode)
}

func TestRegisterOutboxFailureRollsBackUser(t *testing.T) {
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("fail_created_account_outbox")
		cleanUsersTable()
	})
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_created_account_outbox", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*notificationentity.OutboxEventEntity); ok {
			tx.Error = gorm.ErrInvalidData
		}
	}))

	addr := "confirm_rb_" + uuidv7.New().String() + "@ajuda.dev"
	repo := userrepo.NewUserRepository(db, outboxEventRepository)
	_, createErr := repo.CreateUser(&userdomain.UserDomain{
		Name:     "rollback user",
		Email:    addr,
		Password: "123456",
	})
	require.NotNil(t, createErr)

	var count int64
	require.NoError(t, db.Model(&userentity.UserEntity{}).Where("email = ?", addr).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestCreatedAccountWorkerSendsEmailAndHidesFromInbox(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	addr := "confirm_worker_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)

	processCreatedAccountOutbox(t, mailer)

	events := createdAccountEvents(t, user.Id)
	require.Len(t, events, 1)
	assert.Equal(t, notificationdomain.OutboxStatusSent, events[0].Status)
	require.Equal(t, 1, mailer.count())
	code := mailer.lastCode(t)
	assert.NotContains(t, string(events[0].Payload), code)

	page := listNotifications(t, app, validTokenFor(t, user.Id), "")
	assert.Empty(t, page.Data)
}

func TestCreatedAccountWorkerSmtpFailureMarksFailed(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	addr := "confirm_fail_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)

	processCreatedAccountOutbox(t, failingEmailSender{err: errors.New("smtp down")})

	events := createdAccountEvents(t, user.Id)
	require.Len(t, events, 1)
	assert.Equal(t, notificationdomain.OutboxStatusFailed, events[0].Status)
}

func TestVerifyEmailSuccess(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	addr := "confirm_ok_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	processCreatedAccountOutbox(t, mailer)
	code := mailer.lastCode(t)

	resp, err := app.Test(newVerifyEmailRequest(code, validTokenFor(t, user.Id)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body userdto.UserDtoOut
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body.EmailVerified)

	stored, findErr := userRepository.FindById(user.Id)
	require.Nil(t, findErr)
	assert.NotNil(t, stored.EmailVerifiedAt)
}

func TestVerifyEmailWrongCode(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	addr := "confirm_wrong_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	processCreatedAccountOutbox(t, mailer)

	resp, err := app.Test(newVerifyEmailRequest("AAAAAA", validTokenFor(t, user.Id)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var body rest_err.RestErr
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "invalid or expired code", body.Message)

	stored, findErr := userRepository.FindById(user.Id)
	require.Nil(t, findErr)
	assert.Nil(t, stored.EmailVerifiedAt)
}

func TestVerifyEmailUnauthorized(t *testing.T) {
	app := setupApp()
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/v1/user/verify-email", bytes.NewBufferString(`{"code":"AAAAAA"}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestResendVerificationInsertsCreatedAccount(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	addr := "confirm_resend_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	require.Len(t, createdAccountEvents(t, user.Id), 1)

	resp, err := app.Test(newResendVerificationRequest(validTokenFor(t, user.Id)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	assert.Len(t, createdAccountEvents(t, user.Id), 2)
}

func TestResendVerificationAlreadyVerifiedIsNoop(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	app := setupApp()
	user := createUserWithRole(t, "confirm_already_"+uuidv7.New().String()+"@ajuda.dev", userdomain.UserRoleUser)

	resp, err := app.Test(newResendVerificationRequest(validTokenFor(t, user.Id)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	assert.Empty(t, createdAccountEvents(t, user.Id))
}

func TestResendVerificationRateLimit(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Setenv("EMAIL_CODE_RESEND_INTERVAL_SECONDS", "60")
	app := setupApp()
	addr := "confirm_rate_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	token := validTokenFor(t, user.Id)

	resp1, err := app.Test(newResendVerificationRequest(token))
	require.NoError(t, err)
	defer resp1.Body.Close()
	require.Equal(t, fiber.StatusNoContent, resp1.StatusCode)

	resp2, err := app.Test(newResendVerificationRequest(token))
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, fiber.StatusTooManyRequests, resp2.StatusCode)
}

func TestVerifyEmailAttemptRateLimit(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Setenv("EMAIL_CODE_MAX_ATTEMPTS", "2")
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	addr := "confirm_attempts_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	processCreatedAccountOutbox(t, mailer)
	token := validTokenFor(t, user.Id)

	for i := 0; i < 2; i++ {
		resp, err := app.Test(newVerifyEmailRequest("ZZZZZZ", token))
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	}

	resp, err := app.Test(newVerifyEmailRequest("ZZZZZZ", token))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

func TestCreatedAccountWorkerInvalidatesPreviousCode(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	addr := "confirm_invalidate_" + uuidv7.New().String() + "@ajuda.dev"
	_, user := registerConfirmUser(t, app, addr)
	processCreatedAccountOutbox(t, mailer)
	oldCode := mailer.lastCode(t)

	resp, err := app.Test(newResendVerificationRequest(validTokenFor(t, user.Id)))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	processCreatedAccountOutbox(t, mailer)
	newCode := mailer.lastCode(t)
	require.NotEqual(t, oldCode, newCode)

	wrong, err := app.Test(newVerifyEmailRequest(oldCode, validTokenFor(t, user.Id)))
	require.NoError(t, err)
	wrong.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, wrong.StatusCode)

	ok, err := app.Test(newVerifyEmailRequest(newCode, validTokenFor(t, user.Id)))
	require.NoError(t, err)
	defer ok.Body.Close()
	assert.Equal(t, fiber.StatusOK, ok.StatusCode)
}
