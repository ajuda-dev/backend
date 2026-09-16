package notification

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
)

type NotificationService interface {
	List(userId string, status string, page int, limit int) (*notificationdomain.PageableOutboxEvent, *rest_err.RestErr)
	MarkRead(id int64, userId string) (*notificationdomain.OutboxEventDomain, *rest_err.RestErr)
}

type notificationService struct {
	outboxEventRepository notificationrepo.OutboxEventRepository
}

func NewNotificationService(outboxEventRepository notificationrepo.OutboxEventRepository) NotificationService {
	return &notificationService{outboxEventRepository: outboxEventRepository}
}

func (s *notificationService) List(userId string, status string, page int, limit int) (*notificationdomain.PageableOutboxEvent, *rest_err.RestErr) {
	if status == "" {
		status = notificationdomain.NotificationInboxStatusUnread
	}
	switch status {
	case notificationdomain.NotificationInboxStatusUnread, notificationdomain.NotificationInboxStatusRead, notificationdomain.NotificationInboxStatusAll:
	default:
		return nil, rest_err.NewBadRequestValidationError("Invalid query params", []rest_err.Causes{
			{Field: "status", Message: "status must be unread, read or all"},
		})
	}
	return s.outboxEventRepository.FindByUser(userId, notificationrepo.NotificationListFilter{InboxStatus: status}, page, limit)
}

func (s *notificationService) MarkRead(id int64, userId string) (*notificationdomain.OutboxEventDomain, *rest_err.RestErr) {
	return s.outboxEventRepository.MarkRead(id, userId)
}
