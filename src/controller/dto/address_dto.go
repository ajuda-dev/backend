package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type AddressDto struct {
	Id      uint   `json:"id"`
	City    string `json:"city"`
	State   string `json:"state"`
	Street  string `json:"street"`
	ZipCode string `json:"zip_code"`
}

func (a *AddressDto) ToDomain() *domain.AddressDomain {
	return &domain.AddressDomain{
		Id:      a.Id,
		City:    a.City,
		State:   a.State,
		Street:  a.Street,
		ZipCode: a.ZipCode,
	}
}

func (a *AddressDto) FromDomain(address *domain.AddressDomain) *AddressDto {
	return &AddressDto{
		Id:      address.Id,
		City:    address.City,
		State:   address.State,
		Street:  address.Street,
		ZipCode: address.ZipCode,
	}
}