package dto

import (
	userdto "github.com/ajuda-dev/backend/src/controller/identity/dto"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
)

type EventUserDto struct {
	Id      string              `json:"id"`
	EventId string              `json:"event_id"`
	UserId  string              `json:"user_id"`
	Role    string              `json:"role"`
	Status  string              `json:"status"`
	User    *userdto.UserDtoOut `json:"user,omitempty"`
}

func (e EventUserDto) FromDomain(eventUser *eventdomain.EventUserDomain) EventUserDto {
	dtoUser := EventUserDto{
		Id:      eventUser.Id,
		EventId: eventUser.EventId,
		UserId:  eventUser.UserId,
		Role:    eventUser.Role,
		Status:  eventUser.Status,
	}
	if eventUser.User != nil {
		dtoUser.User = (&userdto.UserDtoOut{}).FromDomainUser(eventUser.User)
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
