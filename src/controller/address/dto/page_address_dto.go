package dto

import (
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
)

type PageableAddressDto struct {
	HasNext bool         `json:"has_next"`
	Data    []AddressDto `json:"data"`
}

func (p PageableAddressDto) FromDomain(page addressdomain.PageableAddress) *PageableAddressDto {
	addresses := make([]AddressDto, len(page.Data))
	for i, address := range page.Data {
		addresses[i] = *(&AddressDto{}).FromDomain(address)
	}
	return &PageableAddressDto{
		HasNext: page.HasNext,
		Data:    addresses,
	}
}
