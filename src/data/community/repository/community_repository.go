package repository

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityentity "github.com/ajuda-dev/backend/src/data/community/entity"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type CommunityFilter struct {
	OwnerId string
	Name    string
	City    string
}

type CommunityRepository interface {
	CreateCommunity(community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	FindByName(name string) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	FindById(id string) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	FindAll(filter CommunityFilter, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr)
	Update(id string, community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr)
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

func (c *communityRepository) CreateCommunity(community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity communityentity.CommunityEntity
	communityEntity = *communityEntity.FromDomain(*community)
	communityEntity.Id = uuidv7.New().String()
	if err := c.database.Create(&communityEntity).Error; err != nil {
		return &communitydomain.CommunityDomain{}, rest_err.NewInternalServerError(err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) FindByName(name string) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity communityentity.CommunityEntity
	if err := c.database.Where("name = ?", name).First(&communityEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("community not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting community: " + err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) FindById(id string) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	var communityEntity communityentity.CommunityEntity
	if err := c.database.Preload("Address").Preload("Owner").Where("id = ?", id).First(&communityEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("community not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting community: " + err.Error())
	}
	return communityEntity.ToDomain(), nil
}

func (c *communityRepository) Update(id string, community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	// map (e não struct): `Updates` com struct ignora campos zero, e aqui "vazio"
	// significa "não alterar" — só entram as chaves efetivamente informadas.
	fields := map[string]interface{}{}
	if community.Name != "" {
		fields["name"] = community.Name
	}
	if community.Description != "" {
		fields["description"] = community.Description
	}
	if community.Address.Id != "" {
		fields["address_id"] = community.Address.Id
	}
	result := c.database.Model(&communityentity.CommunityEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error updating community: " + result.Error.Error())
	}
	return c.FindById(id)
}

func (c *communityRepository) SoftDeleteById(id string) *rest_err.RestErr {
	result := c.database.Where("id = ?", id).Delete(&communityentity.CommunityEntity{})
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
	if err := c.database.Model(&communityentity.CommunityEntity{}).
		Where("owner_id = ?", userId).
		Count(&count).Error; err != nil {
		return 0, rest_err.NewInternalServerError("Error counting communities: " + err.Error())
	}
	return count, nil
}

func (c *communityRepository) FindAll(filter CommunityFilter, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr) {
	var communities []communityentity.CommunityEntity
	query := c.database.Model(&communities)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if filter.OwnerId != "" {
		query = query.Where("owner_id = ?", filter.OwnerId)
	}
	name := strings.ToLower(strings.TrimSpace(filter.Name))
	if name != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(name)
		// unaccent nos DOIS lados: o termo passa por ToLower em Go, que preserva acento,
		// então sem unaccent(?) quem digita "são" deixa de achar a base sem acento.
		query = query.Where("unaccent(LOWER(name)) LIKE unaccent(?)", "%"+search+"%")
	}
	city := strings.ToLower(strings.TrimSpace(filter.City))
	if city != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(city)
		// addresses.city já é gravado em minúsculas (AddressEntity.FromDomainAddress),
		// por isso não há LOWER() aqui; só falta remover o acento.
		query = query.Joins("JOIN addresses ON addresses.id = community.address_id").
			Where("unaccent(addresses.city) LIKE unaccent(?)", "%"+search+"%")
	}

	offset := (page - 1) * limit
	result := query.Preload("Address").Preload("Owner").Offset(offset).Limit(limit + 1).Find(&communities)
	if result.Error != nil {
		return &communitydomain.PageableCommunity{}, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(communities) > limit
	if hasNext {
		communities = communities[:limit]
	}
	return &communitydomain.PageableCommunity{
		HasNext: hasNext,
		Data:    communityentity.ToCommunityDomainList(communities),
	}, nil
}
