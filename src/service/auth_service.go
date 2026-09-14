package service

import (
	"os"
	"strconv"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const defaultJWTExpirationHours = 24

func NewAuthService(userRepository repository.UserRepository) AuthService {
	return &authService{
		userRepository: userRepository,
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
	LoginUser(email, password string) (*domain.UserDomain, string, *rest_err.RestErr)
	CreateToken(user *domain.UserDomain) (string, *rest_err.RestErr)
	ValidateToken(tokenString string) (string, *rest_err.RestErr)
}

type authService struct {
	userRepository repository.UserRepository
}

// LoginUser implements AuthService.
func (a *authService) LoginUser(email, password string) (*domain.UserDomain, string, *rest_err.RestErr) {
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

// CreateToken implements AuthService.
func (a *authService) CreateToken(user *domain.UserDomain) (string, *rest_err.RestErr) {
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
