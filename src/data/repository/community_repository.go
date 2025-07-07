package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type CommunityRepository interface {
	CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr)
	FindByName(name string) (*domain.CommunityDomain, *rest_err.RestErr)
	FindAll(address_id int, page int, size int) (*domain.PageableCommunity, *rest_err.RestErr)
}

type communityRepository struct {
	database *gorm.DB
}

func NewCommunityRepository(db *gorm.DB) CommunityRepository {
	return &communityRepository{
		database: db,
	}
}

func (c *communityRepository) CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity entity.CommunityEntity
	communityEntity = *communityEntity.FromDomain(*community)
	if err := c.database.Create(&communityEntity).Error; err != nil {
		return &domain.CommunityDomain{}, rest_err.NewInternalServerError(err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) FindByName(name string) (*domain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity entity.CommunityEntity
	if err := c.database.Where("name = ?", name).First(&communityEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("community not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting community: " + err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) FindAll(address_id int, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr) {
	var communities []entity.CommunityEntity
	query := c.database.Model(&communities)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if address_id != 0 {
		query = query.Where("address_id = ?", address_id)
	}

	offset := (page - 1) * limit
	result := query.Preload("Address").Preload("Owner").Offset(offset).Limit(limit + 1).Find(&communities)
	if result.Error != nil {
		return &domain.PageableCommunity{}, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(communities) > limit
	if hasNext {
		communities = communities[:limit]
	}
	return &domain.PageableCommunity{
		HasNext: hasNext,
		Data:    entity.ToCommunityDomainList(communities),
	}, nil
}
