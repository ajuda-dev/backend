package identity

import (
	"crypto/subtle"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ajuda-dev/backend/src/config/logger"
)

func IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

func RequireAuthSecrets() error {
	jwt := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwt == "" {
		return fmt.Errorf("JWT_SECRET is not configured")
	}
	email := strings.TrimSpace(os.Getenv("EMAIL_CODE_SECRET"))
	if IsProduction() {
		if email == "" || subtle.ConstantTimeCompare([]byte(email), []byte(jwt)) == 1 {
			return fmt.Errorf("EMAIL_CODE_SECRET must be set and distinct from JWT_SECRET")
		}
		if !envBoolTrue("SESSION_COOKIE_SECURE") {
			return fmt.Errorf("SESSION_COOKIE_SECURE must be true in production")
		}
		if !envBoolTrue("OAUTH_COOKIE_SECURE") {
			return fmt.Errorf("OAUTH_COOKIE_SECURE must be true in production")
		}
		return nil
	}
	if email == "" {
		logger.Warn("EMAIL_CODE_SECRET empty; using JWT_SECRET (dev only)")
	}
	return nil
}

func envBoolTrue(key string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return false
	}
	return v
}
