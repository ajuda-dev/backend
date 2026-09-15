package repository

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type EventFilter struct {
	CommunityId        string
	Category           string
	Type               string
	AddressId          string
	City               string
	Upcoming           bool
	UserId             string
	Role               string
	Status             string
	RequesterId        string
	IncludeNonApproved bool
	ApprovalStatus     string
}

type EventRepository interface {
	CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr)
	FindById(id string) (*domain.EventDomain, *rest_err.RestErr)
	FindAll(filter EventFilter, page int, limit int) (*domain.PageableEvent, *rest_err.RestErr)
	SoftDeleteById(id string) *rest_err.RestErr
	CountByOwnerId(userId string) (int64, *rest_err.RestErr)
	UpdateApprovalStatus(id string, current string, status string) (*domain.EventDomain, *rest_err.RestErr)
}

type eventRepository struct {
	database *gorm.DB
	outbox   OutboxEventRepository
}

func NewEventRepository(db *gorm.DB, outbox OutboxEventRepository) EventRepository {
	return &eventRepository{
		database: db,
		outbox:   outbox,
	}
}

func (e *eventRepository) CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr) {
	if event.Status == "" {
		event.Status = domain.EventStatusPending
	}
	txErr := e.database.Transaction(func(tx *gorm.DB) error {
		eventEntity := (&entity.EventEntity{}).FromDomain(*event)
		eventEntity.Id = uuidv7.New().String()
		if err := tx.Create(eventEntity).Error; err != nil {
			return rest_err.NewInternalServerError(err.Error())
		}
		event.Id = eventEntity.Id
		if event.Status == domain.EventStatusPending && event.Community != nil && event.Community.Owner.Id != "" && e.outbox != nil {
			payload, _ := json.Marshal(map[string]string{
				"event_id":     event.Id,
				"title":        event.Title,
				"community_id": event.Community.Id,
				"category":     event.Category,
			})
			outboxEvent := &domain.OutboxEventDomain{
				Type:    domain.OutboxTypeCommunityEventPendingApproval,
				UserId:  event.Community.Owner.Id,
				Payload: payload,
				Status:  domain.OutboxStatusPending,
			}
			if err := e.outbox.Create(tx, outboxEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, toRestErr(txErr)
	}
	return event, nil
}

func (e *eventRepository) FindById(id string) (*domain.EventDomain, *rest_err.RestErr) {
	var eventEntity entity.EventEntity
	query := e.database.Preload("Owner").
		Preload("Address").
		Preload("Community").
		Preload("Community.Owner").
		Preload("Community.Address")
	if err := query.Where("id = ?", id).First(&eventEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, rest_err.NewNotFoundError("event not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting event: " + err.Error())
	}
	return eventEntity.ToDomain(), nil
}

func (e *eventRepository) SoftDeleteById(id string) *rest_err.RestErr {
	result := e.database.Where("id = ?", id).Delete(&entity.EventEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting event: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("event not found")
	}
	return nil
}

func (e *eventRepository) CountByOwnerId(userId string) (int64, *rest_err.RestErr) {
	var count int64
	if err := e.database.Model(&entity.EventEntity{}).
		Where("owner_id = ?", userId).
		Count(&count).Error; err != nil {
		return 0, rest_err.NewInternalServerError("Error counting events: " + err.Error())
	}
	return count, nil
}

func (e *eventRepository) FindAll(filter EventFilter, page int, limit int) (*domain.PageableEvent, *rest_err.RestErr) {
	var events []entity.EventEntity
	query := e.database.Model(&entity.EventEntity{})

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if filter.CommunityId != "" {
		query = query.Where("community_id = ?", filter.CommunityId)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.AddressId != "" {
		query = query.Where("address_id = ?", filter.AddressId)
	}
	if filter.City != "" {
		query = query.Joins("JOIN addresses ON addresses.id = events.address_id").
			Where("addresses.city = ?", strings.ToLower(filter.City))
	}
	if filter.Upcoming {
		query = query.Where("start_at > ?", time.Now())
	}
	if filter.UserId != "" || filter.Role != "" || filter.Status != "" {
		query = query.Joins("JOIN event_users ON event_users.event_id = events.id")
	}
	if filter.UserId != "" {
		query = query.Where("event_users.user_id = ?", filter.UserId)
	}
	if filter.Role != "" {
		query = query.Where("event_users.role = ?", filter.Role)
	}
	if filter.Status != "" {
		query = query.Where("event_users.status = ?", filter.Status)
	}
	if filter.ApprovalStatus != "" {
		query = query.Where("events.status = ?", filter.ApprovalStatus)
	} else if !filter.IncludeNonApproved {
		query = query.Where(
			"events.status = ? OR events.owner_id = ? OR events.community_id IN (SELECT id FROM community WHERE owner_id = ? AND deleted_at IS NULL)",
			domain.EventStatusApproved, filter.RequesterId, filter.RequesterId)
	}
	query = query.Order("start_at")
	query = query.Preload("Owner").
		Preload("Address").
		Preload("Community").
		Preload("Community.Owner").
		Preload("Community.Address")

	offset := (page - 1) * limit
	result := query.Offset(offset).Limit(limit + 1).Find(&events)
	if result.Error != nil {
		return &domain.PageableEvent{}, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(events) > limit
	if hasNext {
		events = events[:limit]
	}
	return &domain.PageableEvent{
		HasNext: hasNext,
		Data:    entity.ToEventDomainList(events),
	}, nil
}

func isValidEventStatusTransition(current string, target string) bool {
	switch current {
	case domain.EventStatusPending:
		return target == domain.EventStatusApproved || target == domain.EventStatusRejected
	case domain.EventStatusRejected:
		return target == domain.EventStatusApproved
	}
	return false
}

func (e *eventRepository) UpdateApprovalStatus(id string, current string, status string) (*domain.EventDomain, *rest_err.RestErr) {
	result := e.database.Model(&entity.EventEntity{}).
		Where("id = ? AND status = ? AND deleted_at IS NULL", id, current).
		Update("status", status)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error updating event status: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid event data",
			[]rest_err.Causes{{
				Field:   "status",
				Message: "invalid status transition from " + current + " to " + status,
			}})
	}
	return e.FindById(id)
}
