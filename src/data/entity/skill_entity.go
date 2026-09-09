package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type SkillEntity struct {
	Id        string `gorm:"primaryKey;type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:idx_skills_name_del,priority:2"`
	// índice composto (name, deleted_at): permite re-registrar o nome após soft delete
	// (a unicidade de nomes ativos é pré-checada no service, não no banco)
	Name string `gorm:"type:varchar(50);not null;uniqueIndex:idx_skills_name_del,priority:1"`
}

func (SkillEntity) TableName() string { return "skills" }

func (s *SkillEntity) FromDomain(skill domain.SkillDomain) *SkillEntity {
	return &SkillEntity{Id: skill.Id, Name: skill.Name}
}

func (s SkillEntity) ToDomain() *domain.SkillDomain {
	return &domain.SkillDomain{Id: s.Id, Name: s.Name}
}

func ToSkillDomainList(entities []SkillEntity) []*domain.SkillDomain {
	var domains []*domain.SkillDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
