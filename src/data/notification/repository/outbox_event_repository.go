package repository

import (
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	notificationentity "github.com/ajuda-dev/backend/src/data/notification/entity"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"gorm.io/gorm"
)

type NotificationListFilter struct {
	InboxStatus string
}

type OutboxEventRepository interface {
	Create(tx *gorm.DB, event *notificationdomain.OutboxEventDomain) *rest_err.RestErr
	FindPending(limit int) ([]notificationdomain.OutboxEventDomain, *rest_err.RestErr)
	UpdateStatus(id int64, from, to string) *rest_err.RestErr
	FindByUser(userId string, filter NotificationListFilter, page int, limit int) (*notificationdomain.PageableOutboxEvent, *rest_err.RestErr)
	MarkRead(id int64, userId string) (*notificationdomain.OutboxEventDomain, *rest_err.RestErr)
}

type outboxEventRepository struct {
	database *gorm.DB
}

func NewOutboxEventRepository(db *gorm.DB) OutboxEventRepository {
	return &outboxEventRepository{database: db}
}

func (r *outboxEventRepository) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.database
}

func (r *outboxEventRepository) Create(tx *gorm.DB, event *notificationdomain.OutboxEventDomain) *rest_err.RestErr {
	var e notificationentity.OutboxEventEntity
	e = *e.FromDomain(*event)
	if e.Status == "" {
		e.Status = notificationdomain.OutboxStatusPending
	}
	if err := r.conn(tx).Create(&e).Error; err != nil {
		return rest_err.NewInternalServerError(err.Error())
	}
	event.Id = e.Id
	event.Status = e.Status
	return nil
}

func (r *outboxEventRepository) FindPending(limit int) ([]notificationdomain.OutboxEventDomain, *rest_err.RestErr) {
	if limit <= 0 {
		limit = 50
	}
	var entities []notificationentity.OutboxEventEntity
	if err := r.database.
		Where("status = ?", notificationdomain.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	result := make([]notificationdomain.OutboxEventDomain, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e.ToDomain())
	}
	return result, nil
}

func (r *outboxEventRepository) UpdateStatus(id int64, from, to string) *rest_err.RestErr {
	result := r.database.Model(&notificationentity.OutboxEventEntity{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	if result.Error != nil {
		return rest_err.NewInternalServerError(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("outbox event not found")
	}
	return nil
}

func (r *outboxEventRepository) FindByUser(userId string, filter NotificationListFilter, page int, limit int) (*notificationdomain.PageableOutboxEvent, *rest_err.RestErr) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	query := r.database.Model(&notificationentity.OutboxEventEntity{}).
		Where("user_id = ? AND type IN ?", userId, notificationdomain.InboxNotificationTypes())
	switch filter.InboxStatus {
	case notificationdomain.NotificationInboxStatusRead:
		query = query.Where("read_at IS NOT NULL")
	case notificationdomain.NotificationInboxStatusAll:
	default:
		query = query.Where("read_at IS NULL")
	}

	var entities []notificationentity.OutboxEventEntity
	offset := (page - 1) * limit
	result := query.Order("created_at DESC, id DESC").Offset(offset).Limit(limit + 1).Find(&entities)
	if result.Error != nil {
		return &notificationdomain.PageableOutboxEvent{}, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(entities) > limit
	if hasNext {
		entities = entities[:limit]
	}
	data := make([]notificationdomain.OutboxEventDomain, 0, len(entities))
	for _, e := range entities {
		data = append(data, *e.ToDomain())
	}
	return &notificationdomain.PageableOutboxEvent{HasNext: hasNext, Data: data}, nil
}

func (r *outboxEventRepository) MarkRead(id int64, userId string) (*notificationdomain.OutboxEventDomain, *rest_err.RestErr) {
	types := notificationdomain.InboxNotificationTypes()
	result := r.database.Model(&notificationentity.OutboxEventEntity{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL AND type IN ?", id, userId, types).
		Update("read_at", gorm.Expr("now()"))
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError(result.Error.Error())
	}

	var stored notificationentity.OutboxEventEntity
	err := r.database.
		Where("id = ? AND user_id = ? AND type IN ?", id, userId, types).
		First(&stored).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, rest_err.NewNotFoundError("notification not found")
	}
	if err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	return stored.ToDomain(), nil
}
