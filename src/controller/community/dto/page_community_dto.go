package dto

import (
	addressdto "github.com/ajuda-dev/backend/src/controller/address/dto"
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

type PageableCommunityDto struct {
	HasNext bool           `json:"has_next"`
	Data    []CommunityDto `json:"data"`
}

type CommunityDto struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Address     *addressdto.AddressDto `json:"address"`
	Owner       *userdto.UserDtoOut    `json:"owner"`
}

func (pc PageableCommunityDto) FromDomain(domain communitydomain.PageableCommunity) *PageableCommunityDto {
	return &PageableCommunityDto{
		HasNext: domain.HasNext,
		Data:    toCommunityDtoSlice(domain.Data),
	}
}

func toCommunityDtoSlice(domains []*communitydomain.CommunityDomain) []CommunityDto {
	dtos := make([]CommunityDto, len(domains))
	for i, d := range domains {
		dtos[i] = CommunityDto{}.FromDomain(d)
	}
	return dtos
}

func (c CommunityDto) FromDomain(community *communitydomain.CommunityDomain) CommunityDto {
	return CommunityDto{
		Id:          community.Id,
		Name:        community.Name,
		Description: community.Description,
		Address:     (&addressdto.AddressDto{}).FromDomain(&community.Address),
		Owner:       (&userdto.UserDtoOut{}).FromDomainUser(&community.Owner),
	}
}
