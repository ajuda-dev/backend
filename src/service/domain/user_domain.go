package domain

type UserDomain struct {
	Id       string
	Name     string
	Email    string
	Password string
	Skills   []SkillDomain
}

type PageableUser struct {
	HasNext bool
	Data    []*UserDomain
}
