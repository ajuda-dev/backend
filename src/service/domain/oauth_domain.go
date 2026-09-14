package domain

const (
	OAuthProviderGithub = "github"
)

type OAuthUserInfo struct {
	Provider         string
	ProviderUserId   string
	ProviderUsername string
	Name             string
	Email            string
}

type OAuthAccountDomain struct {
	Id             string
	UserId         string
	Provider       string
	ProviderUserId string
	Username       string
	Email          string
}
