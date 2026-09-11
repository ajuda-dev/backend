package repository

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type CommunityFilter struct {
	AddressId string
	City      string
}

type CommunityRepository interface {
	CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr)
	FindByName(name string) (*domain.CommunityDomain, *rest_err.RestErr)
	FindById(id string) (*domain.CommunityDomain, *rest_err.RestErr)
	FindAll(filter CommunityFilter, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr)
	SoftDeleteById(id string) *rest_err.RestErr
	CountByOwnerId(userId string) (int64, *rest_err.RestErr)
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
	communityEntity.Id = uuidv7.New().String()
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

func (c *communityRepository) FindById(id string) (*domain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity entity.CommunityEntity
	if err := c.database.Preload("Address").Preload("Owner").Where("id = ?", id).First(&communityEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("community not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting community: " + err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) SoftDeleteById(id string) *rest_err.RestErr {
	result := c.database.Where("id = ?", id).Delete(&entity.CommunityEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting community: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("community not found")
	}
	return nil
}

func (c *communityRepository) CountByOwnerId(userId string) (int64, *rest_err.RestErr) {
	var count int64
	if err := c.database.Model(&entity.CommunityEntity{}).
		Where("owner_id = ?", userId).
		Count(&count).Error; err != nil {
		return 0, rest_err.NewInternalServerError("Error counting communities: " + err.Error())
	}
	return count, nil
}

func (c *communityRepository) FindAll(filter CommunityFilter, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr) {
	var communities []entity.CommunityEntity
	query := c.database.Model(&communities)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if filter.AddressId != "" {
		query = query.Where("address_id = ?", filter.AddressId)
	}
	city := strings.ToLower(strings.TrimSpace(filter.City))
	if city != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(city)
		query = query.Joins("JOIN addresses ON addresses.id = community.address_id").
			Where("addresses.city LIKE ?", "%"+search+"%")
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
