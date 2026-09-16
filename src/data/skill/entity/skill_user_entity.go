package entity

import (
	"time"

	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
)

type SkillUserEntity struct {
	Id        string `gorm:"primaryKey;type:uuid"`
	SkillId   string `gorm:"type:uuid;not null;uniqueIndex:idx_skill_users_skill_user,priority:1"`
	UserId    string `gorm:"type:uuid;not null;uniqueIndex:idx_skill_users_skill_user,priority:2;index:idx_skill_users_user"`
	Level     string `gorm:"type:varchar(20);not null"` // WANT_TO_LEARN | LEARN_AND_TEACH | TEACH
	CreatedAt time.Time
	UpdatedAt time.Time

	Skill SkillEntity           `gorm:"foreignKey:SkillId;references:Id;constraint:OnDelete:CASCADE"`
	User  userentity.UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
}

func (SkillUserEntity) TableName() string { return "skill_users" }

func (e *SkillUserEntity) FromDomain(skillUser skilldomain.SkillUserDomain) *SkillUserEntity {
	return &SkillUserEntity{
		Id:      skillUser.Id,
		SkillId: skillUser.SkillId,
		UserId:  skillUser.UserId,
		Level:   skillUser.Level,
	}
}

func (e SkillUserEntity) ToDomain() *skilldomain.SkillUserDomain {
	skillUser := &skilldomain.SkillUserDomain{
		Id:      e.Id,
		SkillId: e.SkillId,
		UserId:  e.UserId,
		Level:   e.Level,
	}
	if e.Skill.Id != "" {
		skillUser.Skill = e.Skill.ToDomain()
	}
	return skillUser
}

func ToSkillUserDomainList(entities []SkillUserEntity) []*skilldomain.SkillUserDomain {
	var domains []*skilldomain.SkillUserDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
