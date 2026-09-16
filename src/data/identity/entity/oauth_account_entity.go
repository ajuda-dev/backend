package entity

import (
	"time"

	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type OAuthAccountEntity struct {
	Id             string `gorm:"primaryKey;type:uuid"`
	UserId         string `gorm:"type:uuid;not null;uniqueIndex:idx_oauth_accounts_user_provider,priority:1;index:idx_oauth_accounts_user"`
	Provider       string `gorm:"type:varchar(50);not null;uniqueIndex:idx_oauth_accounts_provider_account,priority:1;uniqueIndex:idx_oauth_accounts_user_provider,priority:2"`
	ProviderUserId string `gorm:"type:varchar(191);not null;uniqueIndex:idx_oauth_accounts_provider_account,priority:2"`
	Username       string `gorm:"type:varchar(100)"`
	Email          string `gorm:"type:varchar(100)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	User UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
}

func (OAuthAccountEntity) TableName() string { return "oauth_accounts" }

func (e *OAuthAccountEntity) FromDomain(account userdomain.OAuthAccountDomain) *OAuthAccountEntity {
	return &OAuthAccountEntity{
		Id:             account.Id,
		UserId:         account.UserId,
		Provider:       account.Provider,
		ProviderUserId: account.ProviderUserId,
		Username:       account.Username,
		Email:          account.Email,
	}
}

func (e OAuthAccountEntity) ToDomain() *userdomain.OAuthAccountDomain {
	return &userdomain.OAuthAccountDomain{
		Id:             e.Id,
		UserId:         e.UserId,
		Provider:       e.Provider,
		ProviderUserId: e.ProviderUserId,
		Username:       e.Username,
		Email:          e.Email,
	}
}
