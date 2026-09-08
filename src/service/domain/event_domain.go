package domain

import "time"

const (
	CategoryCommunityEvent = "COMMUNITY_EVENT"
	CategoryMentoring      = "MENTORING"
	CategoryWebinar        = "WEBINAR"

	TypeOnline   = "ONLINE"
	TypeInperson = "INPERSON"
	TypeHybrid   = "HYBRID"
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
}

type PageableEvent struct {
	HasNext bool
	Data    []*EventDomain
}
