package community

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	"github.com/ajuda-dev/backend/src/service/identity"
)

type CommunityUserService interface {
	JoinCommunity(communityId string, userId string) (*communitydomain.CommunityUserDomain, *rest_err.RestErr)
	LeaveCommunity(communityId string, userId string) *rest_err.RestErr
	GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*communitydomain.PageableCommunityMember, *rest_err.RestErr)
	GetUserCommunities(userId string, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr)
}

type communityUserService struct {
	userService             identity.UserService
	communityService        CommunityService
	communityUserRepository communityrepo.CommunityUserRepository
}

func NewCommunityUserService(
	userService identity.UserService,
	communityService CommunityService,
	communityUserRepository communityrepo.CommunityUserRepository) CommunityUserService {
	return &communityUserService{
		userService:             userService,
		communityService:        communityService,
		communityUserRepository: communityUserRepository,
	}
}

func (c *communityUserService) JoinCommunity(communityId string, userId string) (*communitydomain.CommunityUserDomain, *rest_err.RestErr) {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return nil, err
	}
	user, err := identity.AuthenticatedUser(c.userService, userId)
	if err != nil {
		return nil, err
	}
	if err := identity.RequireVerifiedEmail(user); err != nil {
		return nil, err
	}
	return c.communityUserRepository.Create(&communitydomain.CommunityUserDomain{
		CommunityId: communityId,
		UserId:      user.Id,
	})
}

func (c *communityUserService) LeaveCommunity(communityId string, userId string) *rest_err.RestErr {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return err
	}
	return c.communityUserRepository.DeleteByCommunityAndUser(communityId, userId)
}

func (c *communityUserService) GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*communitydomain.PageableCommunityMember, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(c.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return nil, err
	}
	result, err := c.communityUserRepository.FindMembersByCommunity(communityId, page, limit)
	if err != nil {
		return nil, err
	}
	for _, member := range result.Data {
		identity.ApplyVisibilityFilter(member.User, requester)
	}
	return result, nil
}

func (c *communityUserService) GetUserCommunities(userId string, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr) {
	return c.communityUserRepository.FindCommunitiesByUser(userId, page, limit)
}
