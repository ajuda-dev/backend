package service

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/client/oauth"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
)

const (
	oauthStateLength   = 32
	oauthNameMaxLength = 100
)

func NewOAuthService(oauthRegistry *oauth.Registry, oauthAccountRepository repository.OAuthAccountRepository,
	userRepository repository.UserRepository, authService AuthService) OAuthService {
	return &oauthService{
		oauthRegistry:          oauthRegistry,
		oauthAccountRepository: oauthAccountRepository,
		userRepository:         userRepository,
		authService:            authService,
	}
}

type OAuthService interface {
	LoginURL(provider string) (string, string, *rest_err.RestErr)
	Authenticate(provider string, code string) (*domain.UserDomain, string, *rest_err.RestErr)
}

type oauthService struct {
	oauthRegistry          *oauth.Registry
	oauthAccountRepository repository.OAuthAccountRepository
	userRepository         repository.UserRepository
	authService            AuthService
}

func (o *oauthService) LoginURL(providerName string) (string, string, *rest_err.RestErr) {
	provider, providerErr := o.getProvider(providerName)
	if providerErr != nil {
		return "", "", providerErr
	}
	state, err := newOAuthState()
	if err != nil {
		return "", "", rest_err.NewInternalServerError("error generating oauth state: " + err.Error())
	}
	return provider.AuthorizationURL(state), state, nil
}

func (o *oauthService) Authenticate(providerName string, code string) (*domain.UserDomain, string, *rest_err.RestErr) {
	provider, providerErr := o.getProvider(providerName)
	if providerErr != nil {
		return nil, "", providerErr
	}
	if strings.TrimSpace(code) == "" {
		return nil, "", rest_err.NewBadRequestError("authorization code is required")
	}
	accessToken, err := provider.ExchangeCode(code)
	if err != nil {
		return nil, "", err
	}
	userInfo, err := provider.FetchUserInfo(accessToken)
	if err != nil {
		return nil, "", err
	}
	user, err := o.findOrCreateUser(providerName, userInfo)
	if err != nil {
		return nil, "", err
	}
	token, err := o.authService.CreateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (o *oauthService) getProvider(providerName string) (oauth.Provider, *rest_err.RestErr) {
	provider, ok := o.oauthRegistry.Get(providerName)
	if !ok {
		return nil, rest_err.NewBadRequestError("unsupported oauth provider: " + providerName)
	}
	return provider, nil
}

// findOrCreateUser reaproveita o vínculo já existente com o provedor; sem vínculo,
// casa pelo e-mail verificado do provedor e, por fim, cria um usuário novo sem senha.
func (o *oauthService) findOrCreateUser(providerName string, userInfo *domain.OAuthUserInfo) (*domain.UserDomain, *rest_err.RestErr) {
	account, err := o.oauthAccountRepository.FindByProviderAndProviderUserId(providerName, userInfo.ProviderUserId)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}
	if account != nil {
		return o.userRepository.FindById(account.UserId)
	}

	email := strings.ToLower(strings.TrimSpace(userInfo.Email))
	if email == "" {
		return nil, rest_err.NewBadRequestError("oauth provider did not return an email address")
	}

	user, err := o.userRepository.GetUserByEmail(email)
	if err != nil {
		if err.Code != rest_err.NOT_FOUND {
			return nil, err
		}
		user, err = o.createOAuthUser(userInfo, email)
		if err != nil {
			return nil, err
		}
	}

	if _, err := o.oauthAccountRepository.Create(&domain.OAuthAccountDomain{
		UserId:         user.Id,
		Provider:       providerName,
		ProviderUserId: userInfo.ProviderUserId,
		Username:       truncateOAuthValue(userInfo.ProviderUsername),
		Email:          email,
	}); err != nil {
		return nil, err
	}
	return user, nil
}

func (o *oauthService) createOAuthUser(userInfo *domain.OAuthUserInfo, email string) (*domain.UserDomain, *rest_err.RestErr) {
	name := strings.TrimSpace(userInfo.Name)
	if name == "" {
		name = strings.TrimSpace(userInfo.ProviderUsername)
	}
	if name == "" {
		name = email
	}
	now := time.Now()
	return o.userRepository.CreateUser(&domain.UserDomain{
		Name:            truncateOAuthValue(name),
		Email:           email,
		Role:            domain.UserRoleUser,
		EmailVerifiedAt: &now,
	})
}

func newOAuthState() (string, error) {
	buffer := make([]byte, oauthStateLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func truncateOAuthValue(value string) string {
	runes := []rune(value)
	if len(runes) <= oauthNameMaxLength {
		return value
	}
	return string(runes[:oauthNameMaxLength])
}
