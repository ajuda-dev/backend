package validator

import (
	"regexp"
	"strings"
)
var emailRegex = regexp.MustCompile(`^[A-Za-z0-9+_.-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
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