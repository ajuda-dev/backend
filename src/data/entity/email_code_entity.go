package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type EmailCodeEntity struct {
	Id         string `gorm:"primaryKey;type:uuid"`
	UserId     string `gorm:"type:uuid;not null;index:idx_email_codes_user_purpose,priority:1"`
	Purpose    string `gorm:"type:varchar(40);not null;index:idx_email_codes_user_purpose,priority:2"`
	CodeHash   string `gorm:"type:varchar(64);not null"`
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	User UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
}

func (EmailCodeEntity) TableName() string { return "email_codes" }

func (e *EmailCodeEntity) FromDomain(code domain.EmailCodeDomain) *EmailCodeEntity {
	return &EmailCodeEntity{
		Id:         code.Id,
		UserId:     code.UserId,
		Purpose:    code.Purpose,
		CodeHash:   code.CodeHash,
		ExpiresAt:  code.ExpiresAt,
		ConsumedAt: code.ConsumedAt,
	}
}

func (e EmailCodeEntity) ToDomain() *domain.EmailCodeDomain {
	return &domain.EmailCodeDomain{
		Id:         e.Id,
		UserId:     e.UserId,
		Purpose:    e.Purpose,
		CodeHash:   e.CodeHash,
		ExpiresAt:  e.ExpiresAt,
		ConsumedAt: e.ConsumedAt,
	}
}
