package dto

import "github.com/ajuda-dev/backend/src/service/domain"

// Atualização de perfil: só o nome é aceito.
// Email e Password existem apenas para serem recusados com causa explícita
// (o decoder JSON ignora chaves desconhecidas em silêncio).
// Não há campo Role de propósito — cargo não é editável por este endpoint.
type UpdateUserDtoIn struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u *UpdateUserDtoIn) ToDomain() *domain.UserDomain {
	return &domain.UserDomain{
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
	}
}
