package dto

import (
	"github.com/ajuda-dev/backend/src/service/domain"
)

type EventUserDto struct {
	Id      string      `json:"id"`
	EventId string      `json:"event_id"`
	UserId  string      `json:"user_id"`
	Role    string      `json:"role"`
	Status  string      `json:"status"`
	User    *UserDtoOut `json:"user,omitempty"`
}

func (e EventUserDto) FromDomain(eventUser *domain.EventUserDomain) EventUserDto {
	dtoUser := EventUserDto{
		Id:      eventUser.Id,
		EventId: eventUser.EventId,
		UserId:  eventUser.UserId,
		Role:    eventUser.Role,
		Status:  eventUser.Status,
	}
	if eventUser.User != nil {
		dtoUser.User = (&UserDtoOut{}).FromDomainUser(eventUser.User)
	}
	return dtoUser
}

type AddParticipantDto struct {
	UserId string `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateParticipantStatusDto struct {
	Status string `json:"status"`
}
