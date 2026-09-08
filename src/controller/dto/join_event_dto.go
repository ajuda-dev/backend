package dto

import (
	"github.com/ajuda-dev/backend/src/service/domain"
)

type JoinEventDto struct {
	UserId string `json:"user_id"`
}

func (j *JoinEventDto) ToDomain(eventId string) *domain.EventUserDomain {
	return &domain.EventUserDomain{
		EventId: eventId,
		UserId:  j.UserId,
		Role:    domain.RoleAttendee,
		Status:  domain.StatusConfirmed,
	}
}
