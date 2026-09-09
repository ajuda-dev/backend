package domain

const (
	UserRoleUser      = "USER"
	UserRoleModerator = "MODERATOR"
	UserRoleAdmin     = "ADMIN"
)

type UserDomain struct {
	Id       string
	Name     string
	Email    string
	Password string
	Role     string
	Skills   []SkillDomain
}

type PageableUser struct {
	HasNext bool
	Data    []*UserDomain
}
