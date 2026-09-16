package dto

import (
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

type CommunityMemberUserDto struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type CommunityMemberDto struct {
	Id          string                  `json:"id"`
	CommunityId string                  `json:"community_id"`
	UserId      string                  `json:"user_id"`
	User        *CommunityMemberUserDto `json:"user,omitempty"`
}

type PageableCommunityMemberDto struct {
	HasNext bool                 `json:"has_next"`
	Data    []CommunityMemberDto `json:"data"`
}

func (c CommunityMemberDto) FromDomain(communityUser *communitydomain.CommunityUserDomain) CommunityMemberDto {
	dtoMember := CommunityMemberDto{
		Id:          communityUser.Id,
		CommunityId: communityUser.CommunityId,
		UserId:      communityUser.UserId,
	}
	if communityUser.User != nil {
		dtoMember.User = &CommunityMemberUserDto{
			Id:    communityUser.User.Id,
			Name:  communityUser.User.Name,
			Email: communityUser.User.Email,
		}
	}
	return dtoMember
}

func ToCommunityMemberDtoList(communityUsers []*communitydomain.CommunityUserDomain) []CommunityMemberDto {
	dtos := make([]CommunityMemberDto, len(communityUsers))
	for i, cu := range communityUsers {
		dtos[i] = CommunityMemberDto{}.FromDomain(cu)
	}
	return dtos
}

func (p PageableCommunityMemberDto) FromDomain(pageable communitydomain.PageableCommunityMember) *PageableCommunityMemberDto {
	return &PageableCommunityMemberDto{
		HasNext: pageable.HasNext,
		Data:    ToCommunityMemberDtoList(pageable.Data),
	}
}
