package domain

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type CommunityUserDomain struct {
	Id          string
	CommunityId string
	UserId      string
	User        *userdomain.UserDomain
}

type PageableCommunityMember struct {
	HasNext bool
	Data    []*CommunityUserDomain
}
