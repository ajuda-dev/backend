package repository

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type AddressRepository interface {
	CreateAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
	GetAddressById(id string) (*domain.AddressDomain, *rest_err.RestErr)
	SearchAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
}

type addressRepository struct {
	database *gorm.DB
}

func (a *addressRepository) CreateAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	var addressEntity entity.AddressEntity
	addressEntity = *addressEntity.FromDomainAddress(address)
	addressEntity.Id = uuidv7.New().String()
	if err := a.database.Create(&addressEntity).Error; err != nil {
		return &domain.AddressDomain{}, rest_err.NewInternalServerError(err.Error())
	}

	return addressEntity.ToDomainAddress(), nil
}

func (a *addressRepository) SearchAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	var results []entity.AddressEntity
	query := a.database.Model(&entity.AddressEntity{})

	street := strings.ToLower(strings.TrimSpace(address.Street))
	number := strings.TrimSpace(address.Number)
	complement := strings.ToLower(strings.TrimSpace(address.Complement))

	if address.ZipCode != "" {
		query = query.Where("zip_code = ?", address.ZipCode)
	} else {
		if address.City != "" {
			query = query.Where("city = ?", strings.ToLower(strings.TrimSpace(address.City)))
		}
		if address.State != "" {
			query = query.Where("state = ?", strings.ToUpper(strings.TrimSpace(address.State)))
		}
	}
	query = query.Where("street = ?", street).
		Where("number = ?", number).
		Where("complement = ?", complement)
	err := query.Find(&results).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("Address not found")
		}
		return nil, rest_err.NewInternalServerError("Error searching address: " + err.Error())
	}
	if len(results) == 0 {
		return nil, rest_err.NewNotFoundError("Address not found")
	}
	return results[0].ToDomainAddress(), nil
}

func (a *addressRepository) GetAddressById(id string) (*domain.AddressDomain, *rest_err.RestErr) {
	var addressEntity entity.AddressEntity
	if err := a.database.Where("id = ?", id).First(&addressEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("Address not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting address: " + err.Error())
	}
	return addressEntity.ToDomainAddress(), nil
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{
		database: db,
	}
}
