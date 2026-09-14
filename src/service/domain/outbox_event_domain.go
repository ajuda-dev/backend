package domain

import "encoding/json"

const (
	OutboxStatusPending    = "PENDING"
	OutboxStatusProcessing = "PROCESSING"
	OutboxStatusSent       = "SENT"
	OutboxStatusFailed     = "FAILED"
)

const (
	OutboxTypeCommunityEventPendingApproval = "COMMUNITY_EVENT_PENDING_APPROVAL"
	OutboxTypeMentoringInvitePending        = "MENTORING_INVITE_PENDING"
)

type OutboxEventDomain struct {
	Id        int64
	Type      string
	UserId    string
	Payload   json.RawMessage
	Status    string
	CreatedAt string
	UpdatedAt string
}
