package identity

import (
	"html"
	"strings"
	"testing"
)

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

func TestOAuthProviderRedirectHTMLContainsAuthorizationURL(t *testing.T) {
	t.Parallel()
	authorizationURL := "https://github.com/login/oauth/authorize?client_id=abc&state=xyz&redirect_uri=http://127.0.0.1:8080/v1/auth/github/callback"
	page := oauthProviderRedirectHTML(authorizationURL)
	escaped := html.EscapeString(authorizationURL)
	if !strings.Contains(page, `href="`+escaped+`"`) {
		t.Fatalf("esperava href com a URL de autorização no HTML, recebeu %s", page)
	}
	if !strings.Contains(page, "window.location.replace(") {
		t.Fatal("esperava redirect em JavaScript no HTML")
	}
}
