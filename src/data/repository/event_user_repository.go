package repository

import (
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventUserRepository interface {
	CreateOrUpdate(eventUser *domain.EventUserDomain, maxSlots *int) (*domain.EventUserDomain, *rest_err.RestErr)
	FindByEvent(eventId string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr)
	FindByUser(userId string, role string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr)
	FindById(id string) (*domain.EventUserDomain, *rest_err.RestErr)
	UpdateStatus(eventId string, userId string, status string, maxSlots *int) (*domain.EventUserDomain, *rest_err.RestErr)
	CountConfirmedByEvent(eventId string) (int64, *rest_err.RestErr)
}

type eventUserRepository struct {
	database *gorm.DB
}

func NewEventUserRepository(db *gorm.DB) EventUserRepository {
	return &eventUserRepository{
		database: db,
	}
}

func (e *eventUserRepository) FindById(id string) (*domain.EventUserDomain, *rest_err.RestErr) {
	var eventUserEntity entity.EventUserEntity
	if err := e.database.Preload("User").Where("id = ?", id).First(&eventUserEntity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rest_err.NewNotFoundError("participant not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting participant: " + err.Error())
	}
	return eventUserEntity.ToDomain(), nil
}

func (e *eventUserRepository) FindByEvent(eventId string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr) {
	var entities []entity.EventUserEntity
	query := e.database.Preload("User").Where("event_id = ?", eventId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at").Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error getting participants: " + err.Error())
	}
	return entity.ToEventUserDomainList(entities), nil
}

func (e *eventUserRepository) FindByUser(userId string, role string, status string) ([]*domain.EventUserDomain, *rest_err.RestErr) {
	var entities []entity.EventUserEntity
	query := e.database.Preload("User").Where("user_id = ?", userId)
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at").Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error getting participants: " + err.Error())
	}
	return entity.ToEventUserDomainList(entities), nil
}

func (e *eventUserRepository) CountConfirmedByEvent(eventId string) (int64, *rest_err.RestErr) {
	var count int64
	err := e.database.Model(&entity.EventUserEntity{}).
		Joins("JOIN events ON events.id = event_users.event_id").
		Where("events.deleted_at IS NULL AND event_users.event_id = ? AND event_users.status = ?",
			eventId, domain.StatusConfirmed).
		Count(&count).Error
	if err != nil {
		return 0, rest_err.NewInternalServerError("Error counting participants: " + err.Error())
	}
	return count, nil
}

func (e *eventUserRepository) CreateOrUpdate(eventUser *domain.EventUserDomain, maxSlots *int) (*domain.EventUserDomain, *rest_err.RestErr) {
	var result *domain.EventUserDomain
	txErr := e.database.Transaction(func(tx *gorm.DB) error {
		if lockErr := lockEventRow(tx, eventUser.EventId); lockErr != nil {
			return lockErr
		}
		if eventUser.Status == domain.StatusConfirmed {
			if err := checkEventCapacity(tx, eventUser.EventId, maxSlots); err != nil {
				return err
			}
			if eventUser.Role == domain.RoleMentee {
				if err := checkConfirmedMentee(tx, eventUser.EventId, ""); err != nil {
					return err
				}
			}
		}

		var existing entity.EventUserEntity
		queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("event_id = ? AND user_id = ?", eventUser.EventId, eventUser.UserId).
			First(&existing).Error
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return rest_err.NewInternalServerError("Error getting participant: " + queryErr.Error())
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			eventUserEntity := (&entity.EventUserEntity{}).FromDomain(*eventUser)
			eventUserEntity.Id = uuidv7.New().String()
			if err := tx.Create(eventUserEntity).Error; err != nil {
				return rest_err.NewInternalServerError("Error creating participant: " + err.Error())
			}
			result = eventUserEntity.ToDomain()
			return nil
		}

		switch existing.Status {
		case domain.StatusCancelled, domain.StatusRejected:
			if err := tx.Model(&entity.EventUserEntity{}).Where("id = ?", existing.Id).
				Updates(map[string]interface{}{"role": eventUser.Role, "status": eventUser.Status}).Error; err != nil {
				return rest_err.NewInternalServerError("Error updating participant: " + err.Error())
			}
			existing.Role = eventUser.Role
			existing.Status = eventUser.Status
			result = existing.ToDomain()
			return nil
		case domain.StatusRequested:
			return rest_err.NewBadRequestValidationError("Invalid participation data",
				[]rest_err.Causes{{
					Field:   "user_id",
					Message: "user is already invited to this event",
				}})
		default:
			return rest_err.NewBadRequestValidationError("Invalid participation data",
				[]rest_err.Causes{{
					Field:   "user_id",
					Message: "user is already a participant of this event",
				}})
		}
	})
	if txErr != nil {
		return nil, toRestErr(txErr)
	}
	return result, nil
}

func (e *eventUserRepository) UpdateStatus(eventId string, userId string, status string, maxSlots *int) (*domain.EventUserDomain, *rest_err.RestErr) {
	var result *domain.EventUserDomain
	txErr := e.database.Transaction(func(tx *gorm.DB) error {
		if lockErr := lockEventRow(tx, eventId); lockErr != nil {
			return lockErr
		}

		var existing entity.EventUserEntity
		queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("event_id = ? AND user_id = ?", eventId, userId).
			First(&existing).Error
		if queryErr != nil {
			if errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return rest_err.NewNotFoundError("participant not found")
			}
			return rest_err.NewInternalServerError("Error getting participant: " + queryErr.Error())
		}
		if !isValidStatusTransition(existing.Status, status) {
			return invalidStatusTransitionError(existing.Status, status)
		}
		if status == domain.StatusConfirmed {
			if existing.Role == domain.RoleMentee {
				if err := checkConfirmedMentee(tx, eventId, userId); err != nil {
					return err
				}
			}
			if err := checkEventCapacity(tx, eventId, maxSlots); err != nil {
				return err
			}
		}
		if err := tx.Model(&entity.EventUserEntity{}).Where("id = ?", existing.Id).
			Update("status", status).Error; err != nil {
			return rest_err.NewInternalServerError("Error updating participant status: " + err.Error())
		}
		existing.Status = status
		result = existing.ToDomain()
		return nil
	})
	if txErr != nil {
		return nil, toRestErr(txErr)
	}
	return result, nil
}

func lockEventRow(tx *gorm.DB, eventId string) error {
	var eventEntity entity.EventEntity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", eventId).First(&eventEntity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rest_err.NewNotFoundError("event not found")
		}
		return rest_err.NewInternalServerError("Error locking event: " + err.Error())
	}
	return nil
}

func checkEventCapacity(tx *gorm.DB, eventId string, maxSlots *int) error {
	if maxSlots == nil {
		return nil
	}
	var confirmed int64
	if err := tx.Model(&entity.EventUserEntity{}).
		Where("event_id = ? AND status = ?", eventId, domain.StatusConfirmed).
		Count(&confirmed).Error; err != nil {
		return rest_err.NewInternalServerError("Error counting participants: " + err.Error())
	}
	if confirmed >= int64(*maxSlots) {
		return rest_err.NewBadRequestValidationError("Invalid participation data",
			[]rest_err.Causes{{
				Field:   "event_id",
				Message: "event is full",
			}})
	}
	return nil
}

func checkConfirmedMentee(tx *gorm.DB, eventId string, userId string) error {
	var mentees int64
	query := tx.Model(&entity.EventUserEntity{}).
		Where("event_id = ? AND role = ? AND status = ?", eventId, domain.RoleMentee, domain.StatusConfirmed)
	if userId != "" {
		query = query.Where("user_id <> ?", userId)
	}
	if err := query.Count(&mentees).Error; err != nil {
		return rest_err.NewInternalServerError("Error counting mentees: " + err.Error())
	}
	if mentees > 0 {
		return rest_err.NewBadRequestValidationError("Invalid participation data",
			[]rest_err.Causes{{
				Field:   "user_id",
				Message: "event already has a confirmed mentee",
			}})
	}
	return nil
}

func isValidStatusTransition(current string, target string) bool {
	switch current {
	case domain.StatusRequested:
		return target == domain.StatusConfirmed ||
			target == domain.StatusRejected ||
			target == domain.StatusCancelled
	case domain.StatusConfirmed:
		return target == domain.StatusCancelled
	}
	return false
}

func invalidStatusTransitionError(current string, target string) *rest_err.RestErr {
	return rest_err.NewBadRequestValidationError("Invalid participation data",
		[]rest_err.Causes{{
			Field:   "status",
			Message: "invalid status transition from " + current + " to " + target,
		}})
}

func toRestErr(err error) *rest_err.RestErr {
	if err == nil {
		return nil
	}
	if restErr, ok := err.(*rest_err.RestErr); ok {
		return restErr
	}
	return rest_err.NewInternalServerError(err.Error())
}
