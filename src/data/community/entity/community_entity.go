package entity

import (
	"time"

	addressentity "github.com/ajuda-dev/backend/src/data/address/entity"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	"gorm.io/gorm"
)

type CommunityEntity struct {
	Id          string `gorm:"primaryKey;type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt              `gorm:"uniqueIndex:idx_community_name_del,priority:2"`
	Name        string                      `gorm:"not null;uniqueIndex:idx_community_name_del,priority:1"`
	Description string                      `gorm:"not null"`
	OwnerId     string                      `gorm:"type:uuid;not null;index"`
	AddressId   string                      `gorm:"type:uuid;not null;index"`
	Address     addressentity.AddressEntity `gorm:"foreignKey:AddressId;references:Id"`
	Owner       userentity.UserEntity       `gorm:"foreignKey:OwnerId;references:Id"`
}

func (c *CommunityEntity) TableName() string {
	return "community"
}

func (c *CommunityEntity) FromDomain(domain communitydomain.CommunityDomain) *CommunityEntity {

	return &CommunityEntity{
		Id:          domain.Id,
		Name:        domain.Name,
		Description: domain.Description,
		OwnerId:     domain.Owner.Id,
		Owner:       *userentity.FromDomainUser(&domain.Owner),
		AddressId:   domain.Address.Id,
		Address:     *c.Address.FromDomainAddress(&domain.Address),
	}
}

func (c CommunityEntity) ToDomain() *communitydomain.CommunityDomain {
	return &communitydomain.CommunityDomain{
		Id:          c.Id,
		Name:        c.Name,
		Description: c.Description,
		Owner:       *c.Owner.ToDomainUser(),
		Address:     *c.Address.ToDomainAddress(),
	}
}

func ToCommunityDomainList(entities []CommunityEntity) []*communitydomain.CommunityDomain {
	var domains []*communitydomain.CommunityDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
