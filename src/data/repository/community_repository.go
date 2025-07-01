package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type CommunityRepository interface {
	CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr)
}

type communityRepository struct {
	database *gorm.DB
}



func NewCommunityRepository(db *gorm.DB) CommunityRepository {
	return &communityRepository{
		database: db,
	}
}


// createCommunity implements CommunityRepository.
func (c *communityRepository) CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity entity.CommunityEntity
	communityEntity = *communityEntity.FromDomain(*community)
	if err := c.database.Create(&communityEntity).Error; err != nil {
		return &domain.CommunityDomain{}, rest_err.NewInternalServerError(err.Error())
	}
	return communityEntity.ToDomainAddress(), nil
}
