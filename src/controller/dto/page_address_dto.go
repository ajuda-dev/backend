package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type PageableAddressDto struct {
	HasNext bool         `json:"has_next"`
	Data    []AddressDto `json:"data"`
}

func (p PageableAddressDto) FromDomain(page domain.PageableAddress) *PageableAddressDto {
	addresses := make([]AddressDto, len(page.Data))
	for i, address := range page.Data {
		addresses[i] = *(&AddressDto{}).FromDomain(address)
	}
	return &PageableAddressDto{
		HasNext: page.HasNext,
		Data:    addresses,
	}
}
