package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

// Atualização de perfil: nome, resumo e configuração de visibilidade.
// Email e Password existem apenas para serem recusados com causa explícita
// (o decoder JSON ignora chaves desconhecidas em silêncio).
// Não há campo Role nem emailVerified de propósito — cargo e verificação
// de e-mail não são editáveis por este endpoint.
type UpdateUserDtoIn struct {
	Name             string              `json:"name"`
	Email            string              `json:"email"`
	Password         string              `json:"password"`
	Description      string              `json:"description"`
	ConfigVisibility ConfigVisibilityDto `json:"configVisibility"`
}

func (u *UpdateUserDtoIn) ToDomain() *userdomain.UserDomain {
	return &userdomain.UserDomain{
		Name:             u.Name,
		Email:            u.Email,
		Password:         u.Password,
		Description:      u.Description,
		ConfigVisibility: u.ConfigVisibility.ToDomain(),
	}
}
