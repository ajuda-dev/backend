package domain

const (
	LevelWantToLearn   = "WANT_TO_LEARN"
	LevelLearnAndTeach = "LEARN_AND_TEACH"
	LevelTeach         = "TEACH"
)

type SkillUserDomain struct {
	Id      string
	SkillId string
	UserId  string
	Level   string
	Skill   *SkillDomain
}
