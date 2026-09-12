package domain

const (
	UserRoleUser      = "USER"
	UserRoleModerator = "MODERATOR"
	UserRoleAdmin     = "ADMIN"
)

const (
	VisibilityKeyGithub    = "github"
	VisibilityKeyLinkedin  = "linkedin"
	VisibilityKeyOtherlink = "otherlink"
	VisibilityKeyPhoto     = "photo"
	VisibilityKeyEmail     = "email"
	VisibilityKeyPhone     = "phone"
)

type VisibilityConfig struct {
	Value              string `json:"value"`
	ShareWithCommunity bool   `json:"shareWithCommunity"`
}

type ConfigVisibility map[string]VisibilityConfig

type UserDomain struct {
	Id               string
	Name             string
	Email            string
	Password         string
	Role             string
	Description      string
	ConfigVisibility ConfigVisibility
	Skills           []SkillDomain
}

type PageableUser struct {
	HasNext bool
	Data    []*UserDomain
}
