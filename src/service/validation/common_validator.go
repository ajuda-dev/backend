package validator

import (
	"net/url"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[A-Za-z0-9+_.-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
var phoneAllowedChars = regexp.MustCompile(`^\+?[0-9()\-. ]+$`)

func isValidName(name string, isCommunity bool) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}

	if isCommunity {
		return true
	}

	// Regex: apenas letras (incluindo acentos), espaços e apóstrofos
	re := regexp.MustCompile(`^[a-zA-ZÀ-ÿ\s']+$`)

	return re.MatchString(name) && len(name) <= 50
}

func isValidEmail(email string) bool {
	if strings.TrimSpace(email) == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

func isValidHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func isValidPhone(phone string) bool {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" || !phoneAllowedChars.MatchString(trimmed) {
		return false
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, trimmed)
	return len(digits) >= 8 && len(digits) <= 15
}
