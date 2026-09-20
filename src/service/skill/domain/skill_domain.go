package domain

type SkillDomain struct {
	Id        string
	Name      string
	CreatedBy string
}

type PageableSkill struct {
	HasNext bool
	Data    []*SkillDomain
}
