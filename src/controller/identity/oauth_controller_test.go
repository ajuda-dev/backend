package identity

import "testing"

func TestOAuthCanonicalLoginURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requestHost  string
		requestPath  string
		requestQuery string
		callbackURL  string
		wantURL      string
		wantRedirect bool
	}{
		{
			name:         "localhost login with 127.0.0.1 callback",
			requestHost:  "localhost:8080",
			requestPath:  "/v1/auth/github/login",
			callbackURL:  "http://127.0.0.1:8080/v1/auth/github/callback",
			wantURL:      "http://127.0.0.1:8080/v1/auth/github/login",
			wantRedirect: true,
		},
		{
			name:         "already on callback host",
			requestHost:  "127.0.0.1:8080",
			requestPath:  "/v1/auth/github/login",
			callbackURL:  "http://127.0.0.1:8080/v1/auth/github/callback",
			wantRedirect: false,
		},
		{
			name:         "empty request host keeps current request",
			requestHost:  "",
			requestPath:  "/v1/auth/github/login",
			callbackURL:  "http://127.0.0.1:8080/v1/auth/github/callback",
			wantRedirect: false,
		},
		{
			name:         "preserves query string",
			requestHost:  "localhost:8080",
			requestPath:  "/v1/auth/github/login",
			requestQuery: "next=%2Fdashboard",
			callbackURL:  "http://127.0.0.1:8080/v1/auth/github/callback",
			wantURL:      "http://127.0.0.1:8080/v1/auth/github/login?next=%2Fdashboard",
			wantRedirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := oauthCanonicalLoginURL(tt.requestHost, tt.requestPath, tt.requestQuery, tt.callbackURL)
			if ok != tt.wantRedirect {
				t.Fatalf("esperava redirect=%t, recebeu %t (%s)", tt.wantRedirect, ok, got)
			}
			if got != tt.wantURL {
				t.Fatalf("esperava URL %q, recebeu %q", tt.wantURL, got)
			}
		})
	}
}
