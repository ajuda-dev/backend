package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"gorm.io/gorm"
)

type OutboxEventRepository interface {
	Create(tx *gorm.DB, event *domain.OutboxEventDomain) *rest_err.RestErr
	FindPending(limit int) ([]domain.OutboxEventDomain, *rest_err.RestErr)
	UpdateStatus(id int64, from, to string) *rest_err.RestErr
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

func (r *outboxEventRepository) Create(tx *gorm.DB, event *domain.OutboxEventDomain) *rest_err.RestErr {
	var e entity.OutboxEventEntity
	e = *e.FromDomain(*event)
	if e.Status == "" {
		e.Status = domain.OutboxStatusPending
	}
	if err := r.conn(tx).Create(&e).Error; err != nil {
		return rest_err.NewInternalServerError(err.Error())
	}
	event.Id = e.Id
	event.Status = e.Status
	return nil
}

func (r *outboxEventRepository) FindPending(limit int) ([]domain.OutboxEventDomain, *rest_err.RestErr) {
	if limit <= 0 {
		limit = 50
	}
	var entities []entity.OutboxEventEntity
	if err := r.database.
		Where("status = ?", domain.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	result := make([]domain.OutboxEventDomain, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e.ToDomain())
	}
	return result, nil
}

func (r *outboxEventRepository) UpdateStatus(id int64, from, to string) *rest_err.RestErr {
	result := r.database.Model(&entity.OutboxEventEntity{}).
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
