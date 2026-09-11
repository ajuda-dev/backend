package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type PageableCommunityDto struct {
	HasNext bool `json:"has_next"`
	Data    []CommunityDto `json:"data"`
 }

type CommunityDto struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Address     *AddressDto `json:"address"`
	Owner       *UserDtoOut `json:"owner"`
}

func (pc PageableCommunityDto) FromDomain(domain domain.PageableCommunity) *PageableCommunityDto {
	return &PageableCommunityDto{
		HasNext: domain.HasNext,
		Data:    toCommunityDtoSlice(domain.Data),
	}
}

func toCommunityDtoSlice(domains []*domain.CommunityDomain) []CommunityDto {
	dtos := make([]CommunityDto, len(domains))
	for i, d := range domains {
		dtos[i] = CommunityDto{}.FromDomain(d)
	}
	return dtos
}

func (c CommunityDto) FromDomain(community *domain.CommunityDomain) CommunityDto {
	return CommunityDto{
		Id:          community.Id,
		Name:        community.Name,
		Description: community.Description,
		Address:     (&AddressDto{}).FromDomain(&community.Address),
		Owner:       (&UserDtoOut{}).FromDomainUser(&community.Owner),
	}
}
