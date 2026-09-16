package domain

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

const (
	RoleHost     = "HOST"
	RoleMentor   = "MENTOR"
	RoleMentee   = "MENTEE"
	RoleSpeaker  = "SPEAKER"
	RoleAttendee = "ATTENDEE"

	StatusRequested = "REQUESTED"
	StatusConfirmed = "CONFIRMED"
	StatusRejected  = "REJECTED"
	StatusCancelled = "CANCELLED"
)

type EventUserDomain struct {
	Id      string
	EventId string
	UserId  string
	Role    string
	Status  string
	User    *userdomain.UserDomain
}
