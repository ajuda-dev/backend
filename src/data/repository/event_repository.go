package repository

import (
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type EventFilter struct {
	CommunityId string
	Category    string
	Type        string
	AddressId   string
	City        string
	Upcoming    bool
}

type EventRepository interface {
	CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr)
	FindById(id string) (*domain.EventDomain, *rest_err.RestErr)
	FindAll(filter EventFilter, page int, limit int) (*domain.PageableEvent, *rest_err.RestErr)
	SoftDeleteById(id string) *rest_err.RestErr
}

type eventRepository struct {
	database *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{
		database: db,
	}
}

func (e *eventRepository) CreateEvent(event *domain.EventDomain) (*domain.EventDomain, *rest_err.RestErr) {
	eventEntity := (&entity.EventEntity{}).FromDomain(*event)
	eventEntity.Id = uuidv7.New().String()
	if err := e.database.Create(eventEntity).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	event.Id = eventEntity.Id
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
