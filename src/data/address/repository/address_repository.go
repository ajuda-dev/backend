package repository

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type AddressFilter struct {
	City    string
	State   string
	ZipCode string
}

type AddressRepository interface {
	CreateAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
	GetAddressById(id string) (*domain.AddressDomain, *rest_err.RestErr)
	SearchAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
	FindAll(filter AddressFilter, page int, limit int) (*domain.PageableAddress, *rest_err.RestErr)
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

func (a *addressRepository) FindAll(filter AddressFilter, page int, limit int) (*domain.PageableAddress, *rest_err.RestErr) {
	var addresses []entity.AddressEntity
	query := a.database.Model(&entity.AddressEntity{})

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	city := strings.ToLower(strings.TrimSpace(filter.City))
	if city != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(city)
		// addresses.city é gravada em minúsculas, mas preserva acento: sem unaccent nos
		// DOIS lados, quem digita "belém" não acha a base "belem" e vice-versa (plano 16).
		query = query.Where("unaccent(addresses.city) LIKE unaccent(?)", "%"+search+"%")
	}
	if state := strings.ToUpper(strings.TrimSpace(filter.State)); state != "" {
		query = query.Where("addresses.state = ?", state)
	}
	if zip := onlyDigits(filter.ZipCode); zip != "" {
		// a coluna guarda o CEP do ViaCEP ("12345-678"); comparar por dígito aceita
		// com e sem hífen e não depende do formato gravado.
		// A barra dupla é do literal Go: o SQL recebe '\D'.
		query = query.Where("regexp_replace(addresses.zip_code, '\\D', '', 'g') = ?", zip)
	}

	offset := (page - 1) * limit
	result := query.
		Order("addresses.city, addresses.street, addresses.number, addresses.id").
		Offset(offset).Limit(limit + 1).Find(&addresses)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error listing addresses: " + result.Error.Error())
	}

	hasNext := len(addresses) > limit
	if hasNext {
		addresses = addresses[:limit]
	}
	return &domain.PageableAddress{
		HasNext: hasNext,
		Data:    entity.ToAddressDomainList(addresses),
	}, nil
}

// onlyDigits reduz "12345-678", "12345 678" e "12345678" a "12345678".
func onlyDigits(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{
		database: db,
	}
}
