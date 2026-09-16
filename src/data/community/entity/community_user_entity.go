package entity

import (
	"time"

	"github.com/ajuda-dev/backend/src/service/domain"
)

type CommunityUserEntity struct {
	Id          string `gorm:"primaryKey;type:uuid"`
	CommunityId string `gorm:"type:uuid;not null;uniqueIndex:idx_community_users_community_user,priority:1"`
	UserId      string `gorm:"type:uuid;not null;uniqueIndex:idx_community_users_community_user,priority:2;index:idx_community_users_user"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Community CommunityEntity `gorm:"foreignKey:CommunityId;references:Id;constraint:OnDelete:CASCADE"`
	User      UserEntity      `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
}

func (CommunityUserEntity) TableName() string { return "community_users" }

func (e *CommunityUserEntity) FromDomain(cu domain.CommunityUserDomain) *CommunityUserEntity {
	return &CommunityUserEntity{Id: cu.Id, CommunityId: cu.CommunityId, UserId: cu.UserId}
}

func (e CommunityUserEntity) ToDomain() *domain.CommunityUserDomain {
	communityUser := &domain.CommunityUserDomain{Id: e.Id, CommunityId: e.CommunityId, UserId: e.UserId}
	if e.User.Id != "" {
		communityUser.User = e.User.ToDomainUser()
	}
	return communityUser
}

func ToCommunityUserDomainList(entities []CommunityUserEntity) []*domain.CommunityUserDomain {
	var domains []*domain.CommunityUserDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
