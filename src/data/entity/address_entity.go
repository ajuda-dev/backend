package entity

import (
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type AddressEntity struct {
	Id        string    `gorm:"primaryKey;type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	City      string    `gorm:"not null"`
	State     string    `gorm:"not null"`
	ZipCode   string    `gorm:"not null"`
}



func (a *AddressEntity) TableName() string {
	return "addresses"
}

func (a *AddressEntity) ToDomainAddress() *domain.AddressDomain {
	return &domain.AddressDomain{
		Id:      a.Id,
		City:    a.City,
		State:   a.State,
		ZipCode: a.ZipCode,
	}
}

func (a *AddressEntity) FromDomainAddress(address *domain.AddressDomain) *AddressEntity {
	return &AddressEntity{
		Id:      address.Id,
		City:    strings.ToLower(address.City),
		State:   strings.ToUpper(address.State),
		ZipCode: address.ZipCode,
	}
}
