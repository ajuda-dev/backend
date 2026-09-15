package controller_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type recordingEmailSender struct {
	mu       sync.Mutex
	messages []sentEmail
}

type sentEmail struct {
	To      string
	Subject string
	Body    string
}

func (r *recordingEmailSender) Send(to, subject, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, sentEmail{To: to, Subject: subject, Body: body})
	return nil
}

func (r *recordingEmailSender) lastCode(t *testing.T) string {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.messages) == 0 {
		t.Fatal("esperava ao menos um e-mail enviado")
	}
	re := regexp.MustCompile(`[0-9A-Z]{6}`)
	code := re.FindString(r.messages[len(r.messages)-1].Body)
	if code == "" {
		t.Fatalf("não encontrou código no body: %q", r.messages[len(r.messages)-1].Body)
	}
	return code
}

func (r *recordingEmailSender) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.messages)
}

func newForgotPasswordRequest(body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/user/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newResetPasswordRequest(body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/user/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createPasswordUser(t *testing.T, emailAddr, password string) *domain.UserDomain {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "reset user",
		Email:    emailAddr,
		Password: string(hashed),
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	return user
}

func TestForgotPasswordUnknownEmailReturns204(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)

	resp, err := app.Test(newForgotPasswordRequest([]byte(`{"email":"nao-existe@ajuda.dev"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204, recebeu %d", resp.StatusCode)
	}
	if mailer.count() != 0 {
		t.Errorf("não deveria enviar e-mail para conta inexistente, enviou %d", mailer.count())
	}
}

func TestForgotPasswordOAuthUserReturns204WithoutEmail(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)

	_, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "oauth only",
		Email:    "oauth_reset@ajuda.dev",
		Password: "",
	})
	if createErr != nil {
		t.Fatalf("failed to create oauth user: %v", createErr)
	}

	resp, err := app.Test(newForgotPasswordRequest([]byte(`{"email":"oauth_reset@ajuda.dev"}`)))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204, recebeu %d", resp.StatusCode)
	}
	if mailer.count() != 0 {
		t.Errorf("não deveria enviar e-mail para OAuth sem senha, enviou %d", mailer.count())
	}
}

func TestForgotResetPasswordSuccessAllowsLogin(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)

	createPasswordUser(t, "reset_ok@ajuda.dev", "antiga123")

	forgotResp, err := app.Test(newForgotPasswordRequest([]byte(`{"email":"reset_ok@ajuda.dev"}`)))
	if err != nil {
		t.Fatalf("erro no forgot: %v", err)
	}
	defer forgotResp.Body.Close()
	if forgotResp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 no forgot, recebeu %d", forgotResp.StatusCode)
	}

	code := mailer.lastCode(t)
	resetBody := []byte(`{"email":"reset_ok@ajuda.dev","code":"` + code + `","newPassword":"nova456"}`)
	resetResp, err := app.Test(newResetPasswordRequest(resetBody))
	if err != nil {
		t.Fatalf("erro no reset: %v", err)
	}
	defer resetResp.Body.Close()
	if resetResp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resetResp.Body)
		t.Fatalf("esperava 204 no reset, recebeu %d body=%s", resetResp.StatusCode, body)
	}

	oldLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"reset_ok@ajuda.dev","password":"antiga123"}`)))
	if err != nil {
		t.Fatalf("erro no login antigo: %v", err)
	}
	defer oldLogin.Body.Close()
	if oldLogin.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("senha antiga deveria falhar, recebeu %d", oldLogin.StatusCode)
	}

	newLogin, err := app.Test(newUserLoginRequest([]byte(`{"email":"reset_ok@ajuda.dev","password":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro no login novo: %v", err)
	}
	defer newLogin.Body.Close()
	if newLogin.StatusCode != fiber.StatusOK {
		t.Errorf("esperava login com nova senha 200, recebeu %d", newLogin.StatusCode)
	}
}

func TestResetPasswordWrongCodeReturns401(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)

	createPasswordUser(t, "reset_wrong@ajuda.dev", "123456")

	forgotResp, err := app.Test(newForgotPasswordRequest([]byte(`{"email":"reset_wrong@ajuda.dev"}`)))
	if err != nil {
		t.Fatalf("erro no forgot: %v", err)
	}
	defer forgotResp.Body.Close()

	resetResp, err := app.Test(newResetPasswordRequest([]byte(
		`{"email":"reset_wrong@ajuda.dev","code":"AAAAAA","newPassword":"nova456"}`)))
	if err != nil {
		t.Fatalf("erro no reset: %v", err)
	}
	defer resetResp.Body.Close()

	if resetResp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401, recebeu %d", resetResp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resetResp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "invalid or expired code" {
		t.Errorf("esperava message 'invalid or expired code', recebeu '%s'", respBody.Message)
	}
}

func TestForgotPasswordRateLimitReturns429(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Setenv("EMAIL_CODE_RESEND_INTERVAL_SECONDS", "60")
	t.Setenv("EMAIL_CODE_MAX_SENDS_PER_HOUR", "5")

	mailer := &recordingEmailSender{}
	// Novo AuthService lê o env no construtor.
	app := setupAppWithEmail(mailer)

	body := []byte(`{"email":"rate_forgot@ajuda.dev"}`)
	resp1, err := app.Test(newForgotPasswordRequest(body))
	if err != nil {
		t.Fatalf("erro na 1ª request: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != fiber.StatusNoContent {
		t.Fatalf("esperava 204 na 1ª, recebeu %d", resp1.StatusCode)
	}

	resp2, err := app.Test(newForgotPasswordRequest(body))
	if err != nil {
		t.Fatalf("erro na 2ª request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("esperava 429 na 2ª (intervalo), recebeu %d", resp2.StatusCode)
	}
}

func TestResetPasswordAttemptRateLimitReturns429(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	t.Setenv("EMAIL_CODE_MAX_ATTEMPTS", "2")

	mailer := &recordingEmailSender{}
	app := setupAppWithEmail(mailer)
	createPasswordUser(t, "rate_reset@ajuda.dev", "123456")

	_, _ = app.Test(newForgotPasswordRequest([]byte(`{"email":"rate_reset@ajuda.dev"}`)))

	wrong := []byte(`{"email":"rate_reset@ajuda.dev","code":"ZZZZZZ","newPassword":"nova456"}`)
	for i := 0; i < 2; i++ {
		resp, err := app.Test(newResetPasswordRequest(wrong))
		if err != nil {
			t.Fatalf("erro na tentativa %d: %v", i+1, err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("esperava 401 na tentativa %d, recebeu %d", i+1, resp.StatusCode)
		}
	}

	resp, err := app.Test(newResetPasswordRequest(wrong))
	if err != nil {
		t.Fatalf("erro na 3ª tentativa: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("esperava 429 após max attempts, recebeu %d", resp.StatusCode)
	}
}
