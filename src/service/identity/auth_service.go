package identity

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/client/email"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const defaultJWTExpirationHours = 24

func NewAuthService(userRepository userrepo.UserRepository, emailSender email.EmailSender) AuthService {
	if emailSender == nil {
		emailSender = email.NewNoopSender()
	}
	cfg := EmailCodeConfigFromEnv()
	return &authService{
		userRepository: userRepository,
		emailSender:    emailSender,
		emailCodeCfg:   cfg,
		rateLimiter:    newEmailCodeRateLimiter(cfg),
	}
}

// JWTExpiration retorna a validade dos tokens (JWT_EXPIRATION_TIME em horas, padrão 24).
// O cookie de sessão usa o mesmo prazo, para não sobreviver ao token que carrega.
func JWTExpiration() time.Duration {
	expirationHours, err := strconv.Atoi(os.Getenv("JWT_EXPIRATION_TIME"))
	if err != nil || expirationHours <= 0 {
		expirationHours = defaultJWTExpirationHours
	}
	return time.Duration(expirationHours) * time.Hour
}

type AuthService interface {
	LoginUser(email, password string) (*userdomain.UserDomain, string, *rest_err.RestErr)
	CreateToken(user *userdomain.UserDomain) (string, *rest_err.RestErr)
	ValidateToken(tokenString string) (string, *rest_err.RestErr)
	ForgotPassword(emailAddr string) *rest_err.RestErr
	ResetPassword(emailAddr, code, newPassword string) *rest_err.RestErr
	ChangePassword(userId, currentPassword, newPassword string) *rest_err.RestErr
}

type authService struct {
	userRepository userrepo.UserRepository
	emailSender    email.EmailSender
	emailCodeCfg   EmailCodeConfig
	rateLimiter    *emailCodeRateLimiter
}

// LoginUser implements AuthService.
func (a *authService) LoginUser(email, password string) (*userdomain.UserDomain, string, *rest_err.RestErr) {
	user, err := a.userRepository.GetUserByEmail(email)
	if err != nil {
		return nil, "", rest_err.NewUnauthorizedError("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", rest_err.NewUnauthorizedError("invalid credentials")
	}

	token, err := a.CreateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// ForgotPassword implements AuthService.
// Sempre 204 (nil) após o rate limit, para não enumerar e-mails.
func (a *authService) ForgotPassword(emailAddr string) *rest_err.RestErr {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if emailAddr == "" {
		return rest_err.NewBadRequestValidationError("Invalid request", []rest_err.Causes{
			{Field: "email", Message: "Email cannot be empty"},
		})
	}

	now := time.Now()
	if err := a.rateLimiter.AllowSend(emailAddr, now); err != nil {
		return err
	}

	user, err := a.userRepository.GetUserByEmail(emailAddr)
	if err != nil || user == nil || user.Password == "" {
		return nil
	}
	if len(a.emailCodeCfg.Secret) == 0 {
		logger.Error("EMAIL_CODE_SECRET/JWT_SECRET is not configured", nil)
		return nil
	}

	code := generatePasswordResetCode(user.Id, user.Password, now, a.emailCodeCfg.Secret, a.emailCodeCfg.TTLMinutes)
	subject := "Recuperação de senha"
	body := fmt.Sprintf("Seu código de recuperação de senha é: %s \n Este código expira em %d minutos.", code, a.emailCodeCfg.TTLMinutes)
	if sendErr := a.emailSender.Send(emailAddr, subject, body); sendErr != nil {
		logger.Error("failed to send password reset email", sendErr, zap.String("email", emailAddr))
	}
	return nil
}

// ResetPassword implements AuthService.
func (a *authService) ResetPassword(emailAddr, code, newPassword string) *rest_err.RestErr {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	code = strings.ToUpper(strings.TrimSpace(code))

	causes := []rest_err.Causes{}
	if emailAddr == "" {
		causes = append(causes, rest_err.Causes{Field: "email", Message: "Email cannot be empty"})
	}
	if code == "" {
		causes = append(causes, rest_err.Causes{Field: "code", Message: "Code cannot be empty"})
	}
	if len(newPassword) < 6 {
		causes = append(causes, rest_err.Causes{Field: "newPassword", Message: "Password must be at least 6 characters long"})
	}
	if newPassword == "" {
		causes = append(causes, rest_err.Causes{Field: "newPassword", Message: "Password cannot be empty"})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid request", causes)
	}

	now := time.Now()
	if err := a.rateLimiter.AllowAttempt(emailAddr, now); err != nil {
		return err
	}

	invalid := rest_err.NewUnauthorizedError("invalid or expired code")
	user, err := a.userRepository.GetUserByEmail(emailAddr)
	if err != nil || user == nil || user.Password == "" {
		a.rateLimiter.RecordFailedAttempt(emailAddr, now)
		return invalid
	}
	if len(a.emailCodeCfg.Secret) == 0 {
		return rest_err.NewInternalServerError("EMAIL_CODE_SECRET/JWT_SECRET is not configured")
	}
	if !verifyPasswordResetCode(code, user.Id, user.Password, now, a.emailCodeCfg.Secret, a.emailCodeCfg.TTLMinutes) {
		a.rateLimiter.RecordFailedAttempt(emailAddr, now)
		return invalid
	}

	hashed, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		return rest_err.NewInternalServerError(hashErr.Error())
	}
	if updateErr := a.userRepository.UpdatePassword(user.Id, string(hashed)); updateErr != nil {
		return updateErr
	}
	a.rateLimiter.ClearAttempts(emailAddr)
	return nil
}

func (a *authService) ChangePassword(userId, currentPassword, newPassword string) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if currentPassword == "" {
		causes = append(causes, rest_err.Causes{Field: "currentPassword", Message: "Password cannot be empty"})
	}
	if len(newPassword) < 6 {
		causes = append(causes, rest_err.Causes{Field: "newPassword", Message: "Password must be at least 6 characters long"})
	}
	if newPassword == "" {
		causes = append(causes, rest_err.Causes{Field: "newPassword", Message: "Password cannot be empty"})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid request", causes)
	}

	user, err := a.userRepository.FindById(userId)
	if err != nil {
		if err.Code == rest_err.NOT_FOUND {
			return rest_err.NewUnauthorizedError("invalid authenticated user")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return rest_err.NewUnauthorizedError("invalid credentials")
	}

	hashed, hashErr := hashPassword(newPassword)
	if hashErr != nil {
		return rest_err.NewInternalServerError(hashErr.Error())
	}
	return a.userRepository.UpdatePassword(user.Id, hashed)
}

// CreateToken implements AuthService.
func (a *authService) CreateToken(user *userdomain.UserDomain) (string, *rest_err.RestErr) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", rest_err.NewInternalServerError("JWT_SECRET is not configured")
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   user.Id,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(JWTExpiration())),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", rest_err.NewInternalServerError(err.Error())
	}
	return tokenString, nil
}

// ValidateToken implements AuthService.
func (a *authService) ValidateToken(tokenString string) (string, *rest_err.RestErr) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", rest_err.NewInternalServerError("JWT_SECRET is not configured")
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", rest_err.NewUnauthorizedError("invalid or expired token")
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", rest_err.NewUnauthorizedError("invalid or expired token")
	}
	return claims.Subject, nil
}
