package domain

import (
	"encoding/json"
)

const (
	OutboxStatusPending    = "PENDING"
	OutboxStatusProcessing = "PROCESSING"
	OutboxStatusSent       = "SENT"
	OutboxStatusFailed     = "FAILED"
)

const (
	OutboxTypeCommunityEventPendingApproval = "COMMUNITY_EVENT_PENDING_APPROVAL"
	OutboxTypeCommunityEventApproved        = "COMMUNITY_EVENT_APPROVED"
	OutboxTypeCommunityEventRejected        = "COMMUNITY_EVENT_REJECTED"
	OutboxTypeMentoringInvitePending        = "MENTORING_INVITE_PENDING"
	OutboxTypeMentoringInviteAccepted       = "MENTORING_INVITE_ACCEPTED"
	OutboxTypeMentoringInviteRejected       = "MENTORING_INVITE_REJECTED"
	OutboxTypeCreatedAccount                = "CREATED_ACCOUNT"
)

const (
	NotificationInboxStatusUnread = "unread"
	NotificationInboxStatusRead   = "read"
	NotificationInboxStatusAll    = "all"
)

func InboxNotificationTypes() []string {
	return []string{
		OutboxTypeCommunityEventPendingApproval,
		OutboxTypeCommunityEventApproved,
		OutboxTypeCommunityEventRejected,
		OutboxTypeMentoringInvitePending,
		OutboxTypeMentoringInviteAccepted,
		OutboxTypeMentoringInviteRejected,
	}
}

func IsInboxNotificationType(outboxType string) bool {
	for _, t := range InboxNotificationTypes() {
		if t == outboxType {
			return true
		}
	}
	return false
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
