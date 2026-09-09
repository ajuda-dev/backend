package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type AddressDto struct {
	Id         string `json:"id"`
	City       string `json:"city"`
	State      string `json:"state"`
	Street     string `json:"street,omitempty"`
	ZipCode    string `json:"zip_code"`
	Number     string `json:"number,omitempty"`
	Complement string `json:"complement,omitempty"`
}

func (a *AddressDto) ToDomain() *domain.AddressDomain {
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

func (a *AddressDto) FromDomain(address *domain.AddressDomain) *AddressDto {
	return &AddressDto{
		Id:         address.Id,
		City:       address.City,
		State:      address.State,
		Street:     address.Street,
		ZipCode:    address.ZipCode,
		Number:     address.Number,
		Complement: address.Complement,
	}
}
