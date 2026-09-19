package dto

import (
	addressdto "github.com/ajuda-dev/backend/src/controller/address/dto"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type PageableCommunityDto struct {
	HasNext bool           `json:"has_next"`
	Data    []CommunityDto `json:"data"`
}

type CommunityOwnerDto struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type CommunityDto struct {
	Id               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Address          *addressdto.AddressDto `json:"address"`
	Owner            *CommunityOwnerDto     `json:"owner"`
	ConfigVisibility CommunityLinksDto      `json:"configVisibility,omitempty"`
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

func (CommunityOwnerDto) FromDomain(user *userdomain.UserDomain) *CommunityOwnerDto {
	if user == nil {
		return nil
	}
	return &CommunityOwnerDto{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}
}

func (c CommunityDto) FromDomain(community *communitydomain.CommunityDomain) CommunityDto {
	return CommunityDto{
		Id:               community.Id,
		Name:             community.Name,
		Description:      community.Description,
		Address:          (&addressdto.AddressDto{}).FromDomain(&community.Address),
		Owner:            CommunityOwnerDto{}.FromDomain(&community.Owner),
		ConfigVisibility: CommunityLinksDtoFromDomain(community.ConfigVisibility),
	}
}
