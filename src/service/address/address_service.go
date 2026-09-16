package address

import (
	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	addressrepo "github.com/ajuda-dev/backend/src/data/address/repository"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	addressvalidator "github.com/ajuda-dev/backend/src/service/address/validator"
)

type AddressService interface {
	CreateAddress(address *addressdomain.AddressDomain) (*addressdomain.AddressDomain, *rest_err.RestErr)
	GetAddressById(id string) (*addressdomain.AddressDomain, *rest_err.RestErr)
	SearchAddress(address *addressdomain.AddressDomain) (*addressdomain.AddressDomain, *rest_err.RestErr)
	GetAll(filter addressrepo.AddressFilter, page int, limit int) (*addressdomain.PageableAddress, *rest_err.RestErr)
}

type addressService struct {
	addressRepository   addressrepo.AddressRepository
	addressValidator    addressvalidator.AddressValidator
	addressSearchClient client.AddressSearchClient
}

func (a *addressService) CreateAddress(address *addressdomain.AddressDomain) (*addressdomain.AddressDomain, *rest_err.RestErr) {
	search, err := a.SearchAddress(address)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}
	if search != nil {
		return nil, a.duplicatedAddressError()
	}
	viaCep, err := a.addressSearchClient.SearchAddress(*address)
	if err != nil {
		return nil, err
	}
	address.City = viaCep.City
	address.State = viaCep.State
	address.ZipCode = viaCep.ZipCode
	if address.Street == "" {
		address.Street = viaCep.Street
	}
	search, err = a.SearchAddress(address)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}
	if search != nil {
		return nil, a.duplicatedAddressError()
	}
	return a.addressRepository.CreateAddress(address)
}

func (a *addressService) duplicatedAddressError() *rest_err.RestErr {
	return rest_err.NewBadRequestValidationError(
		"Invalid address data",
		[]rest_err.Causes{
			{
				Field:   "address",
				Message: "Address already exists",
			},
		})
}

func (a *addressService) GetAddressById(id string) (*addressdomain.AddressDomain, *rest_err.RestErr) {
	return a.addressRepository.GetAddressById(id)
}

func (a *addressService) SearchAddress(address *addressdomain.AddressDomain) (*addressdomain.AddressDomain, *rest_err.RestErr) {
	err := a.addressValidator.ValidatorSearchAddress(*address)
	if err != nil {
		return nil, err
	}
	return a.addressRepository.SearchAddress(address)
}

func (a *addressService) GetAll(filter addressrepo.AddressFilter, page int, limit int) (*addressdomain.PageableAddress, *rest_err.RestErr) {
	return a.addressRepository.FindAll(filter, page, limit)
}

func NewAddressService(
	addressRepository addressrepo.AddressRepository,
	validatorAddress addressvalidator.AddressValidator,
	addressSearchClient client.AddressSearchClient,
) AddressService {
	return &addressService{
		addressRepository:   addressRepository,
		addressValidator:    validatorAddress,
		addressSearchClient: addressSearchClient,
	}
}
