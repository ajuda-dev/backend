package repository

import (
	"errors"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type EmailCodeRepository interface {
	ReplaceActive(userId, purpose, codeHash string, expiresAt time.Time) *rest_err.RestErr
	FindActive(userId, purpose string, at time.Time) (*userdomain.EmailCodeDomain, *rest_err.RestErr)
	MarkConsumed(id string) *rest_err.RestErr
}

type emailCodeRepository struct {
	database *gorm.DB
}

func NewEmailCodeRepository(db *gorm.DB) EmailCodeRepository {
	return &emailCodeRepository{database: db}
}

func (r *emailCodeRepository) ReplaceActive(userId, purpose, codeHash string, expiresAt time.Time) *rest_err.RestErr {
	txErr := r.database.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&userentity.EmailCodeEntity{}).
			Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", userId, purpose).
			Update("consumed_at", now).Error; err != nil {
			return rest_err.NewInternalServerError(err.Error())
		}
		row := userentity.EmailCodeEntity{
			Id:        uuidv7.New().String(),
			UserId:    userId,
			Purpose:   purpose,
			CodeHash:  codeHash,
			ExpiresAt: expiresAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return rest_err.NewInternalServerError(err.Error())
		}
		return nil
	})
	return toRestErr(txErr)
}

func (r *emailCodeRepository) FindActive(userId, purpose string, at time.Time) (*userdomain.EmailCodeDomain, *rest_err.RestErr) {
	var row userentity.EmailCodeEntity
	err := r.database.
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?", userId, purpose, at).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, rest_err.NewNotFoundError("email code not found")
	}
	if err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}
	return row.ToDomain(), nil
}

func (r *emailCodeRepository) MarkConsumed(id string) *rest_err.RestErr {
	now := time.Now()
	result := r.database.Model(&userentity.EmailCodeEntity{}).
		Where("id = ? AND consumed_at IS NULL", id).
		Update("consumed_at", now)
	if result.Error != nil {
		return rest_err.NewInternalServerError(result.Error.Error())
	}
	return nil
}
