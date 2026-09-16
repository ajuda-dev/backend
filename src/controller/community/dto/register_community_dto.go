package dto

import (
	"github.com/ajuda-dev/backend/src/service/domain"
)

type RegisterCommunityDto struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerId     string `json:"owner_id"`
	AddressId   string `json:"address_id"`
}

func (r *RegisterCommunityDto) ToDomain() *domain.CommunityDomain {
	return &domain.CommunityDomain{
		Name:        r.Name,
		Description: r.Description,
		Owner: domain.UserDomain{
			Id: r.OwnerId,
		},
		Address: domain.AddressDomain{
			Id: r.AddressId,
		},
	}
}
