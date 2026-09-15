package domain

import "time"

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

// VisibleToOthers devolve uma CÓPIA com apenas as entradas compartilhadas
// (shareWithCommunity == true). Entradas não compartilhadas são removidas.
func (c ConfigVisibility) VisibleToOthers() ConfigVisibility {
	visible := make(ConfigVisibility, len(c))
	for key, item := range c {
		if item.ShareWithCommunity {
			visible[key] = item
		}
	}
	return visible
}

type UserDomain struct {
	Id               string
	Name             string
	Email            string
	Password         string
	Role             string
	Description      string
	ConfigVisibility ConfigVisibility
	Skills           []SkillDomain
	EmailVerifiedAt  *time.Time
}

func (u *UserDomain) EmailVerified() bool {
	return u != nil && u.EmailVerifiedAt != nil
}

type PageableUser struct {
	HasNext bool
	Data    []*UserDomain
}
