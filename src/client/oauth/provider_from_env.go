package oauth

import (
	"os"
	"strings"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/service/domain"
	"go.uber.org/zap"
)

var (
	GitHubClientIdEnv     = "GITHUB_CLIENT_ID"
	GitHubClientSecretEnv = "GITHUB_CLIENT_SECRET"
	GitHubCallbackURLEnv  = "GITHUB_CALLBACK_URL"
	GitHubAuthorizeURLEnv = "GITHUB_AUTHORIZE_URL"
	GitHubTokenURLEnv     = "GITHUB_TOKEN_URL"
	GitHubAPIBaseURLEnv   = "GITHUB_API_BASE_URL"
)

const githubCallbackURLDefault = "http://127.0.0.1:8080/v1/auth/github/callback"

// ProvidersFromEnv monta os provedores OAuth habilitados por variáveis de ambiente.
// Um provedor só entra no registry quando está configurado; adicionar google no
// futuro é criar o provider e registrá-lo aqui.
func ProvidersFromEnv() []Provider {
	providers := []Provider{}
	if github, ok := githubProviderFromEnv(); ok {
		providers = append(providers, github)
	}
	return providers
}

func githubProviderFromEnv() (Provider, bool) {
	clientId := strings.TrimSpace(os.Getenv(GitHubClientIdEnv))
	clientSecret := strings.TrimSpace(os.Getenv(GitHubClientSecretEnv))
	if clientId == "" || clientSecret == "" {
		logger.Info("oauth provider disabled",
			zap.String("provider", domain.OAuthProviderGithub),
			zap.String("reason", GitHubClientIdEnv+" and "+GitHubClientSecretEnv+" are not configured"))
		return nil, false
	}
	return NewGithubProvider(
		clientId,
		clientSecret,
		stringOrDefault(os.Getenv(GitHubCallbackURLEnv), githubCallbackURLDefault),
		os.Getenv(GitHubAuthorizeURLEnv),
		os.Getenv(GitHubTokenURLEnv),
		os.Getenv(GitHubAPIBaseURLEnv),
	), true
}
