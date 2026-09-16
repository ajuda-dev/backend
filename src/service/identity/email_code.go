package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	emailCodeCharset       = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	emailCodeLength        = 6
	defaultEmailCodeTTLMin = 5
)

// EmailCodeConfig holds TTL and rate-limit settings for email codes (confirm + forgot/reset).
type EmailCodeConfig struct {
	TTLMinutes            int
	ResendIntervalSeconds int
	MaxSendsPerHour       int
	MaxAttempts           int
	Secret                []byte
}

func EmailCodeConfigFromEnv() EmailCodeConfig {
	cfg := EmailCodeConfig{
		TTLMinutes:            defaultEmailCodeTTLMin,
		ResendIntervalSeconds: 60,
		MaxSendsPerHour:       5,
		MaxAttempts:           5,
	}
	if v := os.Getenv("EMAIL_CODE_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.TTLMinutes = n
		}
	}
	if v := os.Getenv("EMAIL_CODE_RESEND_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ResendIntervalSeconds = n
		}
	}
	if v := os.Getenv("EMAIL_CODE_MAX_SENDS_PER_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxSendsPerHour = n
		}
	}
	if v := os.Getenv("EMAIL_CODE_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxAttempts = n
		}
	}
	secret := strings.TrimSpace(os.Getenv("EMAIL_CODE_SECRET"))
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	cfg.Secret = []byte(secret)
	return cfg
}

func generatePasswordResetCode(userID, passwordHash string, at time.Time, secret []byte, ttlMinutes int) string {
	return codeForWindow(userID, passwordHash, timeWindow(at, ttlMinutes), secret)
}

func verifyPasswordResetCode(code, userID, passwordHash string, at time.Time, secret []byte, ttlMinutes int) bool {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if len(normalized) != emailCodeLength {
		return false
	}
	window := timeWindow(at, ttlMinutes)
	if constantTimeEqualCode(normalized, codeForWindow(userID, passwordHash, window, secret)) {
		return true
	}
	// Janela anterior: código emitido perto do fim da janela ainda vale ~TTL.
	return constantTimeEqualCode(normalized, codeForWindow(userID, passwordHash, window-1, secret))
}

func timeWindow(at time.Time, ttlMinutes int) int64 {
	if ttlMinutes <= 0 {
		ttlMinutes = defaultEmailCodeTTLMin
	}
	return at.Unix() / int64(ttlMinutes*60)
}

func codeForWindow(userID, passwordHash string, window int64, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(userID))
	mac.Write([]byte{0})
	mac.Write([]byte(passwordHash))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(window, 10)))
	sum := mac.Sum(nil)

	code := make([]byte, emailCodeLength)
	for i := 0; i < emailCodeLength; i++ {
		code[i] = emailCodeCharset[int(sum[i])%len(emailCodeCharset)]
	}
	return string(code)
}

func constantTimeEqualCode(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func generateRandomEmailCode() (string, error) {
	code := make([]byte, emailCodeLength)
	max := big.NewInt(int64(len(emailCodeCharset)))
	for i := 0; i < emailCodeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = emailCodeCharset[n.Int64()]
	}
	return string(code), nil
}

func hashEmailConfirmCode(code string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return hex.EncodeToString(mac.Sum(nil))
}
