package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type PageableCommunityDto struct {
	HasNext bool `json:"has_next"`
	Data    []CommunityDto `json:"data"`
 }

type CommunityDto struct {
	Id          uint        `json:"id"`
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
		dtos[i] = CommunityDto{
			Id:          d.Id,
			Name:        d.Name,
			Description: d.Description,
			Address:     (&AddressDto{}).FromDomain(&d.Address),
			Owner:       (&UserDtoOut{}).FromDomainUser(&d.Owner),
		}
	}
	return dtos
}
