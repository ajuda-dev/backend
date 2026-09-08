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

func ToEventUserDtoList(eventUsers []*domain.EventUserDomain) []EventUserDto {
	dtos := make([]EventUserDto, len(eventUsers))
	for i, eu := range eventUsers {
		dtos[i] = EventUserDto{}.FromDomain(eu)
	}
	return dtos
}

type AddParticipantDto struct {
	UserId string `json:"user_id"`
	Role   string `json:"role"`
}

func (a *AddParticipantDto) ToDomain(eventId string) *domain.EventUserDomain {
	return &domain.EventUserDomain{
		EventId: eventId,
		UserId:  a.UserId,
		Role:    a.Role,
	}
}

type UpdateParticipantStatusDto struct {
	Status string `json:"status"`
}

func (u *UpdateParticipantStatusDto) ToDomain(eventId string, userId string) *domain.EventUserDomain {
	return &domain.EventUserDomain{
		EventId: eventId,
		UserId:  userId,
		Status:  u.Status,
	}
}
