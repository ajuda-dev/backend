package repository

import (
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type SkillUserRepository interface {
	Create(skillUser *domain.SkillUserDomain) (*domain.SkillUserDomain, *rest_err.RestErr)
	FindByUser(userId string) ([]*domain.SkillUserDomain, *rest_err.RestErr)
	DeleteByUserAndSkill(userId string, skillId string) *rest_err.RestErr
}

type skillUserRepository struct {
	database *gorm.DB
}

func NewSkillUserRepository(db *gorm.DB) SkillUserRepository {
	return &skillUserRepository{
		database: db,
	}
}

func (s *skillUserRepository) Create(skillUser *domain.SkillUserDomain) (*domain.SkillUserDomain, *rest_err.RestErr) {
	var existing entity.SkillUserEntity
	queryErr := s.database.Where("skill_id = ? AND user_id = ?", skillUser.SkillId, skillUser.UserId).
		First(&existing).Error
	if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
		return nil, rest_err.NewInternalServerError("Error getting skill user: " + queryErr.Error())
	}
	if queryErr == nil {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid skill user data",
			[]rest_err.Causes{{
				Field:   "skill_id",
				Message: "user already has this skill",
			}})
	}

	skillUserEntity := (&entity.SkillUserEntity{}).FromDomain(*skillUser)
	skillUserEntity.Id = uuidv7.New().String()
	if err := s.database.Create(skillUserEntity).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error creating skill user: " + err.Error())
	}
	return skillUserEntity.ToDomain(), nil
}

func (s *skillUserRepository) FindByUser(userId string) ([]*domain.SkillUserDomain, *rest_err.RestErr) {
	var entities []entity.SkillUserEntity
	query := s.database.Preload("Skill").
		Joins("JOIN skills ON skills.id = skill_users.skill_id").
		Where("skill_users.user_id = ?", userId)
	if err := query.Order("skills.name").Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error getting user skills: " + err.Error())
	}
	return entity.ToSkillUserDomainList(entities), nil
}

func (s *skillUserRepository) DeleteByUserAndSkill(userId string, skillId string) *rest_err.RestErr {
	result := s.database.Where("user_id = ? AND skill_id = ?", userId, skillId).
		Delete(&entity.SkillUserEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting skill user: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("skill_user not found")
	}
	return nil
}
