package entity

import (
	"time"

	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"gorm.io/gorm"
)

type SkillEntity struct {
	Id        string `gorm:"primaryKey;type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:idx_skills_name_del,priority:2"`
	// índice composto (name, deleted_at): permite re-registrar o nome após soft delete
	// (a unicidade de nomes ativos é pré-checada no service, não no banco)
	Name      string  `gorm:"type:varchar(50);not null;uniqueIndex:idx_skills_name_del,priority:1"`
	CreatedBy *string `gorm:"type:uuid;index"`
}

func (SkillEntity) TableName() string { return "skills" }

func (s *SkillEntity) FromDomain(skill skilldomain.SkillDomain) *SkillEntity {
	entity := &SkillEntity{Id: skill.Id, Name: skill.Name}
	if skill.CreatedBy != "" {
		createdBy := skill.CreatedBy
		entity.CreatedBy = &createdBy
	}
	return entity
}

func (s SkillEntity) ToDomain() *skilldomain.SkillDomain {
	domain := &skilldomain.SkillDomain{Id: s.Id, Name: s.Name}
	if s.CreatedBy != nil {
		domain.CreatedBy = *s.CreatedBy
	}
	return domain
}

func ToSkillDomainList(entities []SkillEntity) []*skilldomain.SkillDomain {
	var domains []*skilldomain.SkillDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
