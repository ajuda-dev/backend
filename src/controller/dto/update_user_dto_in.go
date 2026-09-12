package dto

import "github.com/ajuda-dev/backend/src/service/domain"

// Atualização de perfil: só o nome é aceito.
// Não há campo Role de propósito — cargo não é editável por este endpoint.
type UpdateUserDtoIn struct {
	Name string `json:"name"`
}

func (u *UpdateUserDtoIn) ToDomain() *domain.UserDomain {
	return &domain.UserDomain{Name: u.Name}
}
