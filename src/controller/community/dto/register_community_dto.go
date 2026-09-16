package dto

import (
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type RegisterCommunityDto struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerId     string `json:"owner_id"`
	AddressId   string `json:"address_id"`
}

func (r *RegisterCommunityDto) ToDomain() *communitydomain.CommunityDomain {
	return &communitydomain.CommunityDomain{
		Name:        r.Name,
		Description: r.Description,
		Owner: userdomain.UserDomain{
			Id: r.OwnerId,
		},
		Address: addressdomain.AddressDomain{
			Id: r.AddressId,
		},
	}
}
