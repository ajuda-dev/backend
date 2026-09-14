package domain

import "time"

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
	Owner       UserDomain
	Community   *CommunityDomain
	Address     *AddressDomain
	MeetingLink string
	MaxSlots    *int
	CreatorRole string
	Status      string
}

type PageableEvent struct {
	HasNext bool
	Data    []*EventDomain
}
