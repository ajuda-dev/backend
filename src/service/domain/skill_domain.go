package domain

type SkillDomain struct {
	Id   string
	Name string
}

type PageableSkill struct {
	HasNext bool
	Data    []*SkillDomain
}
