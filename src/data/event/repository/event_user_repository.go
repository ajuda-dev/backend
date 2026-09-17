package repository

import (
	"encoding/json"
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	evententity "github.com/ajuda-dev/backend/src/data/event/entity"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventUserRepository interface {
	CreateOrUpdate(eventUser *eventdomain.EventUserDomain, maxSlots *int) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	FindByEvent(eventId string, status string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr)
	FindByUser(userId string, role string, status string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr)
	FindById(id string) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	UpdateStatus(eventId string, userId string, status string, comment string, maxSlots *int) (*eventdomain.EventUserDomain, *rest_err.RestErr)
	CountConfirmedByEvent(eventId string) (int64, *rest_err.RestErr)
	CountActiveByUserId(userId string) (int64, *rest_err.RestErr)
}

type eventUserRepository struct {
	database *gorm.DB
	outbox   notificationrepo.OutboxEventRepository
}

func NewEventUserRepository(db *gorm.DB, outbox notificationrepo.OutboxEventRepository) EventUserRepository {
	return &eventUserRepository{
		database: db,
		outbox:   outbox,
	}
}

func (e *eventUserRepository) FindById(id string) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	var eventUserEntity evententity.EventUserEntity
	if err := e.database.Preload("User").Where("id = ?", id).First(&eventUserEntity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rest_err.NewNotFoundError("participant not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting participant: " + err.Error())
	}
	return eventUserEntity.ToDomain(), nil
}

func (e *eventUserRepository) FindByEvent(eventId string, status string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr) {
	var entities []evententity.EventUserEntity
	query := e.database.Preload("User").Where("event_id = ?", eventId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at").Find(&entities).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error getting participants: " + err.Error())
	}
	return evententity.ToEventUserDomainList(entities), nil
}

func (e *eventUserRepository) FindByUser(userId string, role string, status string) ([]*eventdomain.EventUserDomain, *rest_err.RestErr) {
	var entities []evententity.EventUserEntity
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
	return evententity.ToEventUserDomainList(entities), nil
}

func (e *eventUserRepository) CountConfirmedByEvent(eventId string) (int64, *rest_err.RestErr) {
	var count int64
	err := e.database.Model(&evententity.EventUserEntity{}).
		Joins("JOIN events ON events.id = event_users.event_id").
		Where("events.deleted_at IS NULL AND event_users.event_id = ? AND event_users.status = ?",
			eventId, eventdomain.StatusConfirmed).
		Count(&count).Error
	if err != nil {
		return 0, rest_err.NewInternalServerError("Error counting participants: " + err.Error())
	}
	return count, nil
}

func (e *eventUserRepository) CountActiveByUserId(userId string) (int64, *rest_err.RestErr) {
	var count int64
	err := e.database.Model(&evententity.EventUserEntity{}).
		Joins("JOIN events ON events.id = event_users.event_id").
		Where("events.deleted_at IS NULL AND event_users.user_id = ? AND event_users.status IN ?",
			userId, []string{eventdomain.StatusRequested, eventdomain.StatusConfirmed}).
		Count(&count).Error
	if err != nil {
		return 0, rest_err.NewInternalServerError("Error counting participations: " + err.Error())
	}
	return count, nil
}

func (e *eventUserRepository) CreateOrUpdate(eventUser *eventdomain.EventUserDomain, maxSlots *int) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	var result *eventdomain.EventUserDomain
	txErr := e.database.Transaction(func(tx *gorm.DB) error {
		if _, lockErr := lockEventRow(tx, eventUser.EventId); lockErr != nil {
			return lockErr
		}
		if eventUser.Status == eventdomain.StatusConfirmed {
			if err := checkEventCapacity(tx, eventUser.EventId, maxSlots); err != nil {
				return err
			}
			if eventUser.Role == eventdomain.RoleMentee {
				if err := checkConfirmedMentee(tx, eventUser.EventId, ""); err != nil {
					return err
				}
			}
		}

		var existing evententity.EventUserEntity
		queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("event_id = ? AND user_id = ?", eventUser.EventId, eventUser.UserId).
			First(&existing).Error
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return rest_err.NewInternalServerError("Error getting participant: " + queryErr.Error())
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			eventUserEntity := (&evententity.EventUserEntity{}).FromDomain(*eventUser)
			eventUserEntity.Id = uuidv7.New().String()
			if err := tx.Create(eventUserEntity).Error; err != nil {
				return rest_err.NewInternalServerError("Error creating participant: " + err.Error())
			}
			result = eventUserEntity.ToDomain()
			if err := e.insertMentoringInviteOutbox(tx, eventUser); err != nil {
				return err
			}
			return nil
		}

		switch existing.Status {
		case eventdomain.StatusCancelled, eventdomain.StatusRejected:
			if err := tx.Model(&evententity.EventUserEntity{}).Where("id = ?", existing.Id).
				Updates(map[string]interface{}{"role": eventUser.Role, "status": eventUser.Status, "status_comment": nil}).Error; err != nil {
				return rest_err.NewInternalServerError("Error updating participant: " + err.Error())
			}
			existing.Role = eventUser.Role
			existing.Status = eventUser.Status
			existing.StatusComment = nil
			result = existing.ToDomain()
			if err := e.insertMentoringInviteOutbox(tx, eventUser); err != nil {
				return err
			}
			return nil
		case eventdomain.StatusRequested:
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

func (e *eventUserRepository) UpdateStatus(eventId string, userId string, status string, comment string, maxSlots *int) (*eventdomain.EventUserDomain, *rest_err.RestErr) {
	var result *eventdomain.EventUserDomain
	txErr := e.database.Transaction(func(tx *gorm.DB) error {
		eventRow, lockErr := lockEventRow(tx, eventId)
		if lockErr != nil {
			return lockErr
		}

		var existing evententity.EventUserEntity
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
		if status == eventdomain.StatusConfirmed {
			if existing.Role == eventdomain.RoleMentee {
				if err := checkConfirmedMentee(tx, eventId, userId); err != nil {
					return err
				}
			}
			if err := checkEventCapacity(tx, eventId, maxSlots); err != nil {
				return err
			}
		}
		fields := map[string]interface{}{"status": status}
		if status == eventdomain.StatusCancelled || comment == "" {
			fields["status_comment"] = nil
		} else {
			fields["status_comment"] = comment
		}
		if err := tx.Model(&evententity.EventUserEntity{}).Where("id = ?", existing.Id).
			Updates(fields).Error; err != nil {
			return rest_err.NewInternalServerError("Error updating participant status: " + err.Error())
		}
		existing.Status = status
		if status == eventdomain.StatusCancelled || comment == "" {
			existing.StatusComment = nil
		} else {
			existing.StatusComment = &comment
		}
		result = existing.ToDomain()
		if err := e.insertMentoringInviteResponseOutbox(tx, eventRow, userId, status); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, toRestErr(txErr)
	}
	return result, nil
}

func applyMentoringRescheduleStatuses(tx *gorm.DB, outbox notificationrepo.OutboxEventRepository, event *evententity.EventEntity, requesterId string) error {
	if event == nil || event.Category != eventdomain.CategoryMentoring {
		return nil
	}
	var participants []evententity.EventUserEntity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("event_id = ?", event.Id).
		Order("id").
		Find(&participants).Error; err != nil {
		return rest_err.NewInternalServerError("Error getting participants: " + err.Error())
	}

	confirmedUserId := event.OwnerId
	for _, participant := range participants {
		if participant.UserId == requesterId && participant.Status != eventdomain.StatusCancelled {
			confirmedUserId = requesterId
			break
		}
	}

	var pendingUserIds []string
	for _, participant := range participants {
		if participant.Status == eventdomain.StatusCancelled || participant.UserId == confirmedUserId {
			continue
		}
		if participant.Status != eventdomain.StatusConfirmed {
			continue
		}
		if err := updateEventUserStatus(tx, participant.Id, eventdomain.StatusRequested); err != nil {
			return err
		}
		pendingUserIds = append(pendingUserIds, participant.UserId)
	}

	for _, participant := range participants {
		if participant.UserId != confirmedUserId || participant.Status == eventdomain.StatusCancelled {
			continue
		}
		if participant.Status == eventdomain.StatusConfirmed && participant.StatusComment == nil {
			continue
		}
		if err := updateEventUserStatus(tx, participant.Id, eventdomain.StatusConfirmed); err != nil {
			return err
		}
	}

	if outbox == nil || len(pendingUserIds) == 0 {
		return nil
	}
	payload, _ := json.Marshal(map[string]string{
		"event_id": event.Id,
		"category": eventdomain.CategoryMentoring,
	})
	return insertOutboxForUsers(outbox, tx, notificationdomain.OutboxTypeMentoringInvitePending, pendingUserIds, payload)
}

func updateEventUserStatus(tx *gorm.DB, id string, status string) error {
	if err := tx.Model(&evententity.EventUserEntity{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "status_comment": nil}).Error; err != nil {
		return rest_err.NewInternalServerError("Error updating participant status: " + err.Error())
	}
	return nil
}

func (e *eventUserRepository) insertMentoringInviteOutbox(tx *gorm.DB, eventUser *eventdomain.EventUserDomain) error {
	if e.outbox == nil || eventUser.Status != eventdomain.StatusRequested {
		return nil
	}
	payload, _ := json.Marshal(map[string]string{
		"event_id": eventUser.EventId,
		"category": eventdomain.CategoryMentoring,
	})
	return insertOutboxForUsers(e.outbox, tx, notificationdomain.OutboxTypeMentoringInvitePending, []string{eventUser.UserId}, payload)
}

func (e *eventUserRepository) insertMentoringInviteResponseOutbox(tx *gorm.DB, event *evententity.EventEntity, actorId, status string) error {
	if e.outbox == nil || event == nil || event.Category != eventdomain.CategoryMentoring {
		return nil
	}
	outboxType := mentoringInviteResponseOutboxType(status)
	if outboxType == "" {
		return nil
	}
	recipients, err := eventRelatedRecipientIds(tx, event.Id, event.OwnerId, actorId)
	if err != nil {
		return err
	}
	return insertOutboxForUsers(e.outbox, tx, outboxType, recipients, eventUpdateOutboxPayload(event, actorId, status))
}

func mentoringInviteResponseOutboxType(status string) string {
	switch status {
	case eventdomain.StatusConfirmed:
		return notificationdomain.OutboxTypeMentoringInviteAccepted
	case eventdomain.StatusRejected:
		return notificationdomain.OutboxTypeMentoringInviteRejected
	default:
		return ""
	}
}

func lockEventRow(tx *gorm.DB, eventId string) (*evententity.EventEntity, error) {
	var eventEntity evententity.EventEntity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", eventId).First(&eventEntity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rest_err.NewNotFoundError("event not found")
		}
		return nil, rest_err.NewInternalServerError("Error locking event: " + err.Error())
	}
	return &eventEntity, nil
}

func checkEventCapacity(tx *gorm.DB, eventId string, maxSlots *int) error {
	if maxSlots == nil {
		return nil
	}
	var confirmed int64
	if err := tx.Model(&evententity.EventUserEntity{}).
		Where("event_id = ? AND status = ?", eventId, eventdomain.StatusConfirmed).
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
	query := tx.Model(&evententity.EventUserEntity{}).
		Where("event_id = ? AND role = ? AND status = ?", eventId, eventdomain.RoleMentee, eventdomain.StatusConfirmed)
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
	case eventdomain.StatusRequested:
		return target == eventdomain.StatusConfirmed ||
			target == eventdomain.StatusRejected ||
			target == eventdomain.StatusCancelled
	case eventdomain.StatusConfirmed:
		return target == eventdomain.StatusCancelled
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

func eventRelatedRecipientIds(tx *gorm.DB, eventId, ownerId, actorId string) ([]string, error) {
	ids := map[string]struct{}{}
	if ownerId != "" && ownerId != actorId {
		ids[ownerId] = struct{}{}
	}
	var participants []evententity.EventUserEntity
	if err := tx.Select("user_id").Where("event_id = ?", eventId).Find(&participants).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error listing event participants: " + err.Error())
	}
	for _, participant := range participants {
		if participant.UserId != "" && participant.UserId != actorId {
			ids[participant.UserId] = struct{}{}
		}
	}
	recipients := make([]string, 0, len(ids))
	for id := range ids {
		recipients = append(recipients, id)
	}
	return recipients, nil
}

func eventUpdateOutboxPayload(event *evententity.EventEntity, actorId, status string) json.RawMessage {
	data := map[string]string{
		"event_id": event.Id,
		"title":    event.Title,
		"category": event.Category,
		"status":   status,
		"actor_id": actorId,
	}
	if event.CommunityId != nil && *event.CommunityId != "" {
		data["community_id"] = *event.CommunityId
	}
	payload, _ := json.Marshal(data)
	return payload
}

func insertOutboxForUsers(outbox notificationrepo.OutboxEventRepository, tx *gorm.DB, outboxType string, userIds []string, payload json.RawMessage) error {
	if outbox == nil || outboxType == "" {
		return nil
	}
	for _, userId := range userIds {
		if userId == "" {
			continue
		}
		if err := outbox.Create(tx, &notificationdomain.OutboxEventDomain{
			Type:    outboxType,
			UserId:  userId,
			Payload: payload,
			Status:  notificationdomain.OutboxStatusPending,
		}); err != nil {
			return err
		}
	}
	return nil
}
