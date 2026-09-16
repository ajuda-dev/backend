package dto

import "github.com/ajuda-dev/backend/src/service/domain"

// Atualização parcial: campo vazio significa "não alterar".
// Não há OwnerId de propósito — troca de dono não é feita por este endpoint.
type UpdateCommunityDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AddressId   string `json:"address_id"`
}

func (u *UpdateCommunityDto) ToDomain() *domain.CommunityDomain {
	return &domain.CommunityDomain{
		Name:        u.Name,
		Description: u.Description,
		Address: domain.AddressDomain{
			Id: u.AddressId,
		},
	}
}
