package entity

import (
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type CommunityEntity struct {
	gorm.Model
	Id          uint          `gorm:"primaryKey;"`
	Name        string        `gorm:"not null;unique"`
	Description string        `gorm:"not null"`
	OwnerId     string        `gorm:"not null"`
	AddressId   uint          `gorm:"not null"`
	Address     AddressEntity `gorm:"foreignKey:AddressId;references:Id"`
	Owner       UserEntity    `gorm:"foreignKey:OwnerId;references:Id"`
}



func (c *CommunityEntity) TableName() string {
	return "community"
}

func (c *CommunityEntity) FromDomain(domain domain.CommunityDomain) *CommunityEntity {

	return &CommunityEntity{
		Id:          domain.Id,
		Name:        domain.Name,
		Description: domain.Description,
		OwnerId:     domain.Owner.Id,
		Owner:       *FromDomainUser(&domain.Owner),
		AddressId:   domain.Address.Id,
		Address:     *c.Address.FromDomainAddress(&domain.Address),
	}
}


func (c CommunityEntity) ToDomain() *domain.CommunityDomain {
	return &domain.CommunityDomain{
		Id: c.Id,
		Name: c.Name,
		Description: c.Description,
		Owner: *c.Owner.ToDomainUser(),
		Address: *c.Address.ToDomainAddress(),
	}	
}

func ToCommunityDomainList(entities []CommunityEntity) []*domain.CommunityDomain {
	var domains []*domain.CommunityDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
