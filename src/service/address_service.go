package service

import (
	client "github.com/ajuda-dev/backend/src/client/viacep"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type AddressService interface {
	CreateAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
	GetAddressById(id uint) (*domain.AddressDomain, *rest_err.RestErr)
	SearchAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
}

type addressService struct {
	addressRepository repository.AddressRepository
	addressValidator 	validator.AddressValidator
	addressSearchClient client.AddressSearchClient
}


func (a *addressService) CreateAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	search, err := a.SearchAddress(address)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, err
	}	
	if search != nil {
	return search, rest_err.NewBadRequestValidationError(
		"Invalid address data",
		[]rest_err.Causes{
			{
				Field:   "address",
				Message: "Address already exists",
			},
		})
	}
	address ,err = a.addressSearchClient.SearchAddress(*address)
	if err != nil {
		return nil, rest_err.NewInternalServerError("error search address: " + err.Error())
	}
	return a.addressRepository.CreateAddress(address)
}


func (a *addressService) GetAddressById(id uint) (*domain.AddressDomain, *rest_err.RestErr) {
	panic("unimplemented")
}


func (a *addressService) SearchAddress(address *domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	err := a.addressValidator.ValidatorSearchAddress(*address)
	if err != nil {	
		return nil, err
	}	
	return a.addressRepository.SearchAddress(address)
}

func NewAddressService(
	addressRepository repository.AddressRepository,
	validatorAddress validator.AddressValidator,
	addressSearchClient client.AddressSearchClient,
	) AddressService {
	return &addressService{
		addressRepository: addressRepository,
		addressValidator: validatorAddress,
		addressSearchClient: addressSearchClient,
	}
}
