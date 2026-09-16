package repository

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	skillentity "github.com/ajuda-dev/backend/src/data/skill/entity"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type SkillFilter struct {
	Name string
}

type SkillRepository interface {
	CreateSkill(skill *skilldomain.SkillDomain) (*skilldomain.SkillDomain, *rest_err.RestErr)
	FindByName(name string) (*skilldomain.SkillDomain, *rest_err.RestErr)
	FindById(id string) (*skilldomain.SkillDomain, *rest_err.RestErr)
	FindAll(filter SkillFilter, page int, limit int) (*skilldomain.PageableSkill, *rest_err.RestErr)
	SoftDeleteById(id string) *rest_err.RestErr
	UpdateName(id string, name string) (*skilldomain.SkillDomain, *rest_err.RestErr)
}

type skillRepository struct {
	database *gorm.DB
}

func NewSkillRepository(db *gorm.DB) SkillRepository {
	return &skillRepository{
		database: db,
	}
}

func (s *skillRepository) CreateSkill(skill *skilldomain.SkillDomain) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	skillEntity := (&skillentity.SkillEntity{}).FromDomain(*skill)
	skillEntity.Id = uuidv7.New().String()
	if err := s.database.Create(skillEntity).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	return skillEntity.ToDomain(), nil
}

func (s *skillRepository) FindByName(name string) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	var skillEntity skillentity.SkillEntity
	if err := s.database.Where("name = ?", name).First(&skillEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("skill not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting skill: " + err.Error())
	}
	return skillEntity.ToDomain(), nil
}

func (s *skillRepository) FindById(id string) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	var skillEntity skillentity.SkillEntity
	if err := s.database.Where("id = ?", id).First(&skillEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("skill not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting skill: " + err.Error())
	}
	return skillEntity.ToDomain(), nil
}

func (s *skillRepository) FindAll(filter SkillFilter, page int, limit int) (*skilldomain.PageableSkill, *rest_err.RestErr) {
	var skills []skillentity.SkillEntity
	query := s.database.Model(&skillentity.SkillEntity{})

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if filter.Name != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(filter.Name)
		query = query.Where("name LIKE ?", search+"%")
	}

	offset := (page - 1) * limit
	result := query.Order("name").Offset(offset).Limit(limit + 1).Find(&skills)
	if result.Error != nil {
		return &skilldomain.PageableSkill{}, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(skills) > limit
	if hasNext {
		skills = skills[:limit]
	}
	return &skilldomain.PageableSkill{
		HasNext: hasNext,
		Data:    skillentity.ToSkillDomainList(skills),
	}, nil
}

func (s *skillRepository) SoftDeleteById(id string) *rest_err.RestErr {
	txErr := s.database.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ?", id).Delete(&skillentity.SkillEntity{})
		if result.Error != nil {
			return rest_err.NewInternalServerError("Error deleting skill: " + result.Error.Error())
		}
		if result.RowsAffected == 0 {
			return rest_err.NewNotFoundError("skill not found")
		}
		if err := tx.Where("skill_id = ?", id).Delete(&skillentity.SkillUserEntity{}).Error; err != nil {
			return rest_err.NewInternalServerError("Error deleting skill users: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return toRestErr(txErr)
	}
	return nil
}

func (s *skillRepository) UpdateName(id string, name string) (*skilldomain.SkillDomain, *rest_err.RestErr) {
	result := s.database.Model(&skillentity.SkillEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("name", name)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error updating skill: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return nil, rest_err.NewNotFoundError("skill not found")
	}
	return &skilldomain.SkillDomain{Id: id, Name: name}, nil
}
