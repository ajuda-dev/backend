package identity

import (
	"strings"
	"testing"
)

func TestRequireAuthSecrets(t *testing.T) {
	tests := []struct {
		name          string
		appEnv        string
		jwtSecret     string
		emailSecret   string
		sessionSecure string
		oauthSecure   string
		wantErrSubstr string
	}{
		{
			name:          "production empty EMAIL_CODE_SECRET",
			appEnv:        "production",
			jwtSecret:     "a",
			emailSecret:   "",
			sessionSecure: "true",
			oauthSecure:   "true",
			wantErrSubstr: "EMAIL_CODE_SECRET must be set and distinct from JWT_SECRET",
		},
		{
			name:          "production secrets equal",
			appEnv:        "production",
			jwtSecret:     "same",
			emailSecret:   "same",
			sessionSecure: "true",
			oauthSecure:   "true",
			wantErrSubstr: "EMAIL_CODE_SECRET must be set and distinct from JWT_SECRET",
		},
		{
			name:          "production cookies false",
			appEnv:        "production",
			jwtSecret:     "jwt",
			emailSecret:   "email",
			sessionSecure: "false",
			oauthSecure:   "false",
			wantErrSubstr: "SESSION_COOKIE_SECURE must be true in production",
		},
		{
			name:          "production oauth cookie false",
			appEnv:        "production",
			jwtSecret:     "jwt",
			emailSecret:   "email",
			sessionSecure: "true",
			oauthSecure:   "false",
			wantErrSubstr: "OAUTH_COOKIE_SECURE must be true in production",
		},
		{
			name:          "production ok",
			appEnv:        "production",
			jwtSecret:     "jwt",
			emailSecret:   "email",
			sessionSecure: "true",
			oauthSecure:   "true",
		},
		{
			name:          "development empty APP_ENV allows fallback",
			appEnv:        "",
			jwtSecret:     "test",
			emailSecret:   "",
			sessionSecure: "false",
			oauthSecure:   "false",
		},
		{
			name:          "empty JWT_SECRET in development",
			appEnv:        "",
			jwtSecret:     "",
			emailSecret:   "",
			sessionSecure: "false",
			oauthSecure:   "false",
			wantErrSubstr: "JWT_SECRET is not configured",
		},
		{
			name:          "empty JWT_SECRET in production",
			appEnv:        "production",
			jwtSecret:     "",
			emailSecret:   "email",
			sessionSecure: "true",
			oauthSecure:   "true",
			wantErrSubstr: "JWT_SECRET is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("JWT_SECRET", tt.jwtSecret)
			t.Setenv("EMAIL_CODE_SECRET", tt.emailSecret)
			t.Setenv("SESSION_COOKIE_SECURE", tt.sessionSecure)
			t.Setenv("OAUTH_COOKIE_SECURE", tt.oauthSecure)

			err := RequireAuthSecrets()
			if tt.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("esperava nil, recebeu %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava erro contendo %q", tt.wantErrSubstr)
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("erro %q não contém %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	t.Run("empty is development", func(t *testing.T) {
		t.Setenv("APP_ENV", "")
		if IsProduction() {
			t.Fatal("APP_ENV vazio não deveria ser production")
		}
	})
	t.Run("development", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		if IsProduction() {
			t.Fatal("development não deveria ser production")
		}
	})
	t.Run("production case insensitive", func(t *testing.T) {
		t.Setenv("APP_ENV", "Production")
		if !IsProduction() {
			t.Fatal("Production deveria ser production")
		}
	})
}
