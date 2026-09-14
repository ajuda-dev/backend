package oauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

const (
	githubAuthorizeURLDefault = "https://github.com/login/oauth/authorize"
	githubTokenURLDefault     = "https://github.com/login/oauth/access_token"
	githubAPIBaseURLDefault   = "https://api.github.com"
	githubScope               = "user:email"
	githubRequestTimeout      = 10 * time.Second
)

type githubProvider struct {
	clientId     string
	clientSecret string
	callbackUrl  string
	authorizeUrl string
	tokenUrl     string
	apiBaseUrl   string
	httpClient   *http.Client
}

func NewGithubProvider(clientId, clientSecret, callbackUrl, authorizeUrl, tokenUrl, apiBaseUrl string) Provider {
	return &githubProvider{
		clientId:     clientId,
		clientSecret: clientSecret,
		callbackUrl:  callbackUrl,
		authorizeUrl: stringOrDefault(authorizeUrl, githubAuthorizeURLDefault),
		tokenUrl:     stringOrDefault(tokenUrl, githubTokenURLDefault),
		apiBaseUrl:   strings.TrimSuffix(stringOrDefault(apiBaseUrl, githubAPIBaseURLDefault), "/"),
		httpClient:   &http.Client{Timeout: githubRequestTimeout},
	}
}

func (g *githubProvider) Name() string {
	return domain.OAuthProviderGithub
}

func (g *githubProvider) AuthorizationURL(state string) string {
	params := url.Values{}
	params.Set("client_id", g.clientId)
	params.Set("redirect_uri", g.callbackUrl)
	params.Set("scope", githubScope)
	params.Set("state", state)
	return g.authorizeUrl + "?" + params.Encode()
}

type githubTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (g *githubProvider) ExchangeCode(code string) (string, *rest_err.RestErr) {
	form := url.Values{}
	form.Set("client_id", g.clientId)
	form.Set("client_secret", g.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", g.callbackUrl)

	request, err := http.NewRequest(http.MethodPost, g.tokenUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return "", rest_err.NewInternalServerError("error building github token request: " + err.Error())
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return "", rest_err.NewInternalServerError("error calling github token endpoint: " + err.Error())
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", rest_err.NewInternalServerError("error reading github token response: " + err.Error())
	}

	var tokenResponse githubTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", rest_err.NewInternalServerError("error decoding github token response: " + err.Error())
	}
	if tokenResponse.Error != "" || tokenResponse.AccessToken == "" {
		return "", rest_err.NewUnauthorizedError("invalid or expired github authorization code")
	}
	return tokenResponse.AccessToken, nil
}

type githubUserResponse struct {
	Id    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type githubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *githubProvider) FetchUserInfo(accessToken string) (*domain.OAuthUserInfo, *rest_err.RestErr) {
	var user githubUserResponse
	if err := g.getJSON("/user", accessToken, &user); err != nil {
		return nil, err
	}
	if user.Id == 0 {
		return nil, rest_err.NewInternalServerError("github api did not return the user id")
	}

	var emails []githubEmailResponse
	if err := g.getJSON("/user/emails", accessToken, &emails); err != nil {
		return nil, err
	}
	email := githubVerifiedEmail(emails)
	if email == "" {
		return nil, rest_err.NewBadRequestError("github account has no verified email address")
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Login
	}

	return &domain.OAuthUserInfo{
		Provider:         domain.OAuthProviderGithub,
		ProviderUserId:   strconv.FormatInt(user.Id, 10),
		ProviderUsername: user.Login,
		Name:             name,
		Email:            email,
	}, nil
}

func (g *githubProvider) getJSON(path string, accessToken string, target interface{}) *rest_err.RestErr {
	request, err := http.NewRequest(http.MethodGet, g.apiBaseUrl+path, nil)
	if err != nil {
		return rest_err.NewInternalServerError("error building github api request: " + err.Error())
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return rest_err.NewInternalServerError("error calling github api: " + err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return rest_err.NewUnauthorizedError("github api returned status " + strconv.Itoa(response.StatusCode))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return rest_err.NewInternalServerError("error reading github api response: " + err.Error())
	}
	if err := json.Unmarshal(body, &target); err != nil {
		return rest_err.NewInternalServerError("error decoding github api response: " + err.Error())
	}
	return nil
}

func githubVerifiedEmail(emails []githubEmailResponse) string {
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email
		}
	}
	for _, email := range emails {
		if email.Verified {
			return email.Email
		}
	}
	return ""
}

func stringOrDefault(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
