package oauth

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

// Provider é o contrato que qualquer provedor OAuth precisa cumprir: cada
// implementação (github, google, ...) devolve os dados do usuário já
// normalizados em userdomain.OAuthUserInfo, então o restante do sistema não
// conhece detalhes de nenhum provedor específico.
type Provider interface {
	Name() string
	AuthorizationURL(state string) string
	ExchangeCode(code string) (string, *rest_err.RestErr)
	FetchUserInfo(accessToken string) (*userdomain.OAuthUserInfo, *rest_err.RestErr)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{
		providers: make(map[string]Provider, len(providers)),
	}
	for _, provider := range providers {
		registry.providers[provider.Name()] = provider
	}
	return registry
}

func (r *Registry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}
