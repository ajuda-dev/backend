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

const (
	NotificationInboxStatusUnread = "unread"
	NotificationInboxStatusRead   = "read"
	NotificationInboxStatusAll    = "all"
)

func InboxNotificationTypes() []string {
	return []string{
		OutboxTypeCommunityEventPendingApproval,
		OutboxTypeMentoringInvitePending,
	}
}

type OutboxEventDomain struct {
	Id        int64
	Type      string
	UserId    string
	Payload   json.RawMessage
	Status    string
	CreatedAt string
	UpdatedAt string
	ReadAt    string
}

type PageableOutboxEvent struct {
	HasNext bool
	Data    []OutboxEventDomain
}
