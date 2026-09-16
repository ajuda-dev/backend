package dto

import (
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

type CommunityUserDto struct {
	Id          string `json:"id"`
	CommunityId string `json:"community_id"`
	UserId      string `json:"user_id"`
}

func (c CommunityUserDto) FromDomain(communityUser *communitydomain.CommunityUserDomain) CommunityUserDto {
	return CommunityUserDto{
		Id:          communityUser.Id,
		CommunityId: communityUser.CommunityId,
		UserId:      communityUser.UserId,
	}
}
