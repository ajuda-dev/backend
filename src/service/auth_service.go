package service

import (
	"os"
	"strconv"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/golang-jwt/jwt/v5"
)

func NewAuthService() AuthService {
	return &authService{}
}

type AuthService interface {
	CreateToken(user *domain.UserDomain) (string, *rest_err.RestErr)
}

type authService struct{}

// CreateToken implements AuthService.
func (a *authService) CreateToken(user *domain.UserDomain) (string, *rest_err.RestErr) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", rest_err.NewInternalServerError("JWT_SECRET is not configured")
	}

	expirationHours, err := strconv.Atoi(os.Getenv("JWT_EXPIRATION_TIME"))
	if err != nil || expirationHours <= 0 {
		expirationHours = 24
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   user.Id,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expirationHours) * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", rest_err.NewInternalServerError(err.Error())
	}
	return tokenString, nil
}
