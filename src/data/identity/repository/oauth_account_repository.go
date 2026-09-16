package repository

import (
	"errors"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type OAuthAccountRepository interface {
	Create(account *domain.OAuthAccountDomain) (*domain.OAuthAccountDomain, *rest_err.RestErr)
	FindByProviderAndProviderUserId(provider string, providerUserId string) (*domain.OAuthAccountDomain, *rest_err.RestErr)
}

type oauthAccountRepository struct {
	database *gorm.DB
}

func NewOAuthAccountRepository(db *gorm.DB) OAuthAccountRepository {
	return &oauthAccountRepository{
		database: db,
	}
}

func (o *oauthAccountRepository) Create(account *domain.OAuthAccountDomain) (*domain.OAuthAccountDomain, *rest_err.RestErr) {
	accountEntity := (&entity.OAuthAccountEntity{}).FromDomain(*account)
	accountEntity.Id = uuidv7.New().String()
	if err := o.database.Create(accountEntity).Error; err != nil {
		return nil, rest_err.NewInternalServerError("Error creating oauth account: " + err.Error())
	}
	return accountEntity.ToDomain(), nil
}

func (o *oauthAccountRepository) FindByProviderAndProviderUserId(provider string, providerUserId string) (*domain.OAuthAccountDomain, *rest_err.RestErr) {
	var accountEntity entity.OAuthAccountEntity
	if err := o.database.
		Where("provider = ? AND provider_user_id = ?", provider, providerUserId).
		First(&accountEntity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rest_err.NewNotFoundError("OAuth account not found")
		}
		return nil, rest_err.NewInternalServerError("Error getting oauth account: " + err.Error())
	}
	return accountEntity.ToDomain(), nil
}
