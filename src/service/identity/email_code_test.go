package identity

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndVerifyPasswordResetCode(t *testing.T) {
	secret := []byte("test-secret")
	now := time.Unix(1_700_000_000, 0)
	code := generatePasswordResetCode("user-1", "$2a$hashed", now, secret, 5)
	if len(code) != 6 {
		t.Fatalf("esperava código de 6 chars, recebeu %q", code)
	}
	if !verifyPasswordResetCode(code, "user-1", "$2a$hashed", now, secret, 5) {
		t.Fatal("código válido deveria passar")
	}
	if !verifyPasswordResetCode(code, "user-1", "$2a$hashed", now.Add(4*time.Minute), secret, 5) {
		t.Fatal("código deveria valer dentro da janela")
	}
	if verifyPasswordResetCode(code, "user-1", "$2a$other", now, secret, 5) {
		t.Fatal("hash diferente deveria invalidar")
	}
	if verifyPasswordResetCode("AAAAAA", "user-1", "$2a$hashed", now, secret, 5) {
		t.Fatal("código errado não deveria passar")
	}
	if !verifyPasswordResetCode(code, "user-1", "$2a$hashed", now.Add(6*time.Minute), secret, 5) {
		// janela anterior ainda aceita (código emitido no fim da janela anterior)
		t.Fatal("deveria aceitar janela anterior")
	}
	if verifyPasswordResetCode(code, "user-1", "$2a$hashed", now.Add(11*time.Minute), secret, 5) {
		t.Fatal("após duas janelas o código deveria expirar")
	}
}

func TestGenerateRandomEmailCode(t *testing.T) {
	code, err := generateRandomEmailCode()
	if err != nil {
		t.Fatalf("generateRandomEmailCode: %v", err)
	}
	if len(code) != emailCodeLength {
		t.Fatalf("esperava código de %d chars, recebeu %q", emailCodeLength, code)
	}
	for _, ch := range code {
		if !strings.ContainsRune(emailCodeCharset, ch) {
			t.Fatalf("char %q fora do charset", ch)
		}
	}
}

func TestHashEmailConfirmCodeNormalizesInput(t *testing.T) {
	secret := []byte("secret")
	upper := hashEmailConfirmCode("AB12CD", secret)
	lower := hashEmailConfirmCode("ab12cd", secret)
	if upper != lower {
		t.Fatal("hash deveria ser case-insensitive")
	}
	if len(upper) != 64 {
		t.Fatalf("esperava hash hex de 64 chars, recebeu %d", len(upper))
	}
}
