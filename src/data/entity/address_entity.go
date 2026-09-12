package entity

import (
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type AddressEntity struct {
	Id         string `gorm:"primaryKey;type:uuid"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt
	City       string `gorm:"not null"`
	State      string `gorm:"not null"`
	ZipCode    string `gorm:"not null"`
	Street     string `gorm:"type:varchar(200);not null;default:''"`
	Number     string `gorm:"type:varchar(20);not null;default:''"`
	Complement string `gorm:"type:varchar(60);not null;default:''"`
}

func (a *AddressEntity) TableName() string {
	return "addresses"
}

func (a *AddressEntity) ToDomainAddress() *domain.AddressDomain {
	return &domain.AddressDomain{
		Id:         a.Id,
		City:       a.City,
		State:      a.State,
		Street:     a.Street,
		ZipCode:    a.ZipCode,
		Number:     a.Number,
		Complement: a.Complement,
	}
}

func (a *AddressEntity) FromDomainAddress(address *domain.AddressDomain) *AddressEntity {
	return &AddressEntity{
		Id:         address.Id,
		City:       strings.ToLower(address.City),
		State:      strings.ToUpper(address.State),
		ZipCode:    address.ZipCode,
		Street:     strings.ToLower(strings.TrimSpace(address.Street)),
		Number:     strings.TrimSpace(address.Number),
		Complement: strings.ToLower(strings.TrimSpace(address.Complement)),
	}
}

func ToAddressDomainList(entities []AddressEntity) []*domain.AddressDomain {
	addresses := make([]*domain.AddressDomain, len(entities))
	for i, e := range entities {
		addresses[i] = e.ToDomainAddress()
	}
	return addresses
}
