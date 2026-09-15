package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type NotificationService interface {
	List(userId string, status string, page int, limit int) (*domain.PageableOutboxEvent, *rest_err.RestErr)
	MarkRead(id int64, userId string) (*domain.OutboxEventDomain, *rest_err.RestErr)
}

type notificationService struct {
	outboxEventRepository repository.OutboxEventRepository
}

func NewNotificationService(outboxEventRepository repository.OutboxEventRepository) NotificationService {
	return &notificationService{outboxEventRepository: outboxEventRepository}
}

func (s *notificationService) List(userId string, status string, page int, limit int) (*domain.PageableOutboxEvent, *rest_err.RestErr) {
	if status == "" {
		status = domain.NotificationInboxStatusUnread
	}
	switch status {
	case domain.NotificationInboxStatusUnread, domain.NotificationInboxStatusRead, domain.NotificationInboxStatusAll:
	default:
		return nil, rest_err.NewBadRequestValidationError("Invalid query params", []rest_err.Causes{
			{Field: "status", Message: "status must be unread, read or all"},
		})
	}
	return s.outboxEventRepository.FindByUser(userId, repository.NotificationListFilter{InboxStatus: status}, page, limit)
}

func (s *notificationService) MarkRead(id int64, userId string) (*domain.OutboxEventDomain, *rest_err.RestErr) {
	return s.outboxEventRepository.MarkRead(id, userId)
}
