package domain

import (
	"time"

	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

const (
	CategoryCommunityEvent = "COMMUNITY_EVENT"
	CategoryMentoring      = "MENTORING"
	CategoryWebinar        = "WEBINAR"

	TypeOnline   = "ONLINE"
	TypeInperson = "INPERSON"
	TypeHybrid   = "HYBRID"

	EventStatusPending  = "PENDING"
	EventStatusApproved = "APPROVED"
	EventStatusRejected = "REJECTED"
)

type EventDomain struct {
	Id          string
	Category    string
	Type        string
	Title       string
	Description string
	StartAt     time.Time
	DurationMin int
	Owner       userdomain.UserDomain
	Community   *communitydomain.CommunityDomain
	Address     *addressdomain.AddressDomain
	MeetingLink string
	MaxSlots    *int
	CreatorRole string
	Status      string
	Comment     string
}

type PageableEvent struct {
	HasNext bool
	Data    []*EventDomain
}
