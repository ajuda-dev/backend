package entity

import (
	"strings"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type AddressEntity struct {
	gorm.Model
	Id      uint   `gorm:"primaryKey;autoIncrement"`
	City    string `gorm:"not null"`
	State   string `gorm:"not null"`
	ZipCode string `gorm:"not null"`
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