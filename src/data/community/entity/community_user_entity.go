package entity

import (
	"time"

	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

type CommunityUserEntity struct {
	Id          string `gorm:"primaryKey;type:uuid"`
	CommunityId string `gorm:"type:uuid;not null;uniqueIndex:idx_community_users_community_user,priority:1"`
	UserId      string `gorm:"type:uuid;not null;uniqueIndex:idx_community_users_community_user,priority:2;index:idx_community_users_user"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Community CommunityEntity       `gorm:"foreignKey:CommunityId;references:Id;constraint:OnDelete:CASCADE"`
	User      userentity.UserEntity `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
}

func (CommunityUserEntity) TableName() string { return "community_users" }

func (e *CommunityUserEntity) FromDomain(cu communitydomain.CommunityUserDomain) *CommunityUserEntity {
	return &CommunityUserEntity{Id: cu.Id, CommunityId: cu.CommunityId, UserId: cu.UserId}
}

func (e CommunityUserEntity) ToDomain() *communitydomain.CommunityUserDomain {
	communityUser := &communitydomain.CommunityUserDomain{Id: e.Id, CommunityId: e.CommunityId, UserId: e.UserId}
	if e.User.Id != "" {
		communityUser.User = e.User.ToDomainUser()
	}
	return communityUser
}

func ToCommunityUserDomainList(entities []CommunityUserEntity) []*communitydomain.CommunityUserDomain {
	var domains []*communitydomain.CommunityUserDomain
	for _, e := range entities {
		domains = append(domains, e.ToDomain())
	}
	return domains
}
