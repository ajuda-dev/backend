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

	CommentKindReschedule = "RESCHEDULE"
	CommentKindReject     = "REJECT"
	CommentKindCancel     = "CANCEL"
	CommentKindNote       = "NOTE"
)

type EventUserDomain struct {
	Id                string
	EventId           string
	UserId            string
	Role              string
	Status            string
	StatusComment     string
	StatusCommentKind string
	User              *userdomain.UserDomain
}
