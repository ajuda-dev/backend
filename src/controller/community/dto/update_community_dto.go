package dto

import (
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

// Atualização parcial: campo vazio significa "não alterar".
// Não há OwnerId de propósito — troca de dono não é feita por este endpoint.
type UpdateCommunityDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AddressId   string `json:"address_id"`
}

func (u *UpdateCommunityDto) ToDomain() *communitydomain.CommunityDomain {
	return &communitydomain.CommunityDomain{
		Name:        u.Name,
		Description: u.Description,
		Address: addressdomain.AddressDomain{
			Id: u.AddressId,
		},
	}
}
