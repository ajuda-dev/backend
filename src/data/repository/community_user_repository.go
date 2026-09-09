package repository

import (
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type CommunityUserRepository interface {
	Create(communityUser *domain.CommunityUserDomain) (*domain.CommunityUserDomain, *rest_err.RestErr)
	DeleteByCommunityAndUser(communityId string, userId string) *rest_err.RestErr
	CountByCommunity(communityId string) (int64, *rest_err.RestErr)
	CountByUserId(userId string) (int64, *rest_err.RestErr)
}

type communityUserRepository struct {
	database *gorm.DB
}

func NewCommunityUserRepository(db *gorm.DB) CommunityUserRepository {
	return &communityUserRepository{
		database: db,
	}
}

func (c *communityUserRepository) Create(communityUser *domain.CommunityUserDomain) (*domain.CommunityUserDomain, *rest_err.RestErr) {
	var existing entity.CommunityUserEntity
	queryErr := c.database.Where("community_id = ? AND user_id = ?", communityUser.CommunityId, communityUser.UserId).
		First(&existing).Error
	if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
		return nil, rest_err.NewInternalServerError("Error getting membership: " + queryErr.Error())
	}
	if queryErr == nil {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid membership data",
			[]rest_err.Causes{{
				Field:   "community_id",
				Message: "user is already a member of this community",
			}})
	}

	communityUserEntity := (&entity.CommunityUserEntity{}).FromDomain(*communityUser)
	communityUserEntity.Id = uuidv7.New().String()
	if err := c.database.Create(communityUserEntity).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error creating membership: " + err.Error())
	}
	return communityUserEntity.ToDomain(), nil
}

func (c *communityUserRepository) DeleteByCommunityAndUser(communityId string, userId string) *rest_err.RestErr {
	result := c.database.Where("community_id = ? AND user_id = ?", communityId, userId).
		Delete(&entity.CommunityUserEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting membership: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("membership not found")
	}
	return nil
}

func (c *communityUserRepository) CountByCommunity(communityId string) (int64, *rest_err.RestErr) {
	var count int64
	if err := c.database.Model(&entity.CommunityUserEntity{}).
		Where("community_id = ?", communityId).
		Count(&count).Error; err != nil {
		return 0, rest_err.NewInternalServerError("Error counting memberships: " + err.Error())
	}
	return count, nil
}

func (c *communityUserRepository) CountByUserId(userId string) (int64, *rest_err.RestErr) {
	var count int64
	if err := c.database.Model(&entity.CommunityUserEntity{}).
		Joins("JOIN community ON community.id = community_users.community_id").
		Where("community.deleted_at IS NULL AND community_users.user_id = ?", userId).
		Count(&count).Error; err != nil {
		return 0, rest_err.NewInternalServerError("Error counting memberships: " + err.Error())
	}
	return count, nil
}
