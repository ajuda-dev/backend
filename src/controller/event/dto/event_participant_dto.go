package dto

import (
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
)

type ParticipantUserDto struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type EventParticipantDto struct {
	Id      string              `json:"id"`
	EventId string              `json:"event_id"`
	UserId  string              `json:"user_id"`
	Role    string              `json:"role"`
	Status  string              `json:"status"`
	User    *ParticipantUserDto `json:"user,omitempty"`
}

func (e EventParticipantDto) FromDomain(eventUser *eventdomain.EventUserDomain) EventParticipantDto {
	dtoParticipant := EventParticipantDto{
		Id:      eventUser.Id,
		EventId: eventUser.EventId,
		UserId:  eventUser.UserId,
		Role:    eventUser.Role,
		Status:  eventUser.Status,
	}
	if eventUser.User != nil {
		dtoParticipant.User = &ParticipantUserDto{
			Id:    eventUser.User.Id,
			Name:  eventUser.User.Name,
			Email: eventUser.User.Email,
		}
	}
	return dtoParticipant
}

func ToEventParticipantDtoList(eventUsers []*eventdomain.EventUserDomain) []EventParticipantDto {
	dtos := make([]EventParticipantDto, len(eventUsers))
	for i, eu := range eventUsers {
		dtos[i] = EventParticipantDto{}.FromDomain(eu)
	}
	return dtos
}
